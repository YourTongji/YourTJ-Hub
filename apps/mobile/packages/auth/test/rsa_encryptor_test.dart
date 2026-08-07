import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';

import 'package:auth/src/rsa/rsa_encryptor.dart';
import 'package:pointycastle/asn1.dart';
import 'package:pointycastle/asymmetric/api.dart';
import 'package:pointycastle/digests/sha256.dart';

/// RSA-OAEP 加密互操作测试。
///
/// 密钥对由 openssl 生成(test/fixtures/test_private.pem + test_public.pem),
/// 与 Go `rsa.EncryptOAEP(sha256.New(), ...)` 使用相同参数
/// (RFC 8017,EM 带前导 0x00)。这里用私钥做裸 RSA 解密,再校验
/// EME-OAEP 结构(0x00 || maskedSeed || maskedDB),并复算掩码还原明文,
/// 证明加密输出能被标准 OAEP 解码器(Go 后端)正确解密。
void main() {
  final File pubFile = File('test/fixtures/test_public.pem');
  final File privFile = File('test/fixtures/test_private.pem');

  setUpAll(() {
    if (!pubFile.existsSync() || !privFile.existsSync()) {
      fail('缺少测试密钥对 fixture: 请运行 '
          'openssl genrsa -out test/fixtures/test_private.pem 2048 && '
          'openssl rsa -in test/fixtures/test_private.pem -pubout '
          '-out test/fixtures/test_public.pem');
    }
  });

  group('RsaEncryptor', () {
    test('密文可被标准 OAEP 解码还原明文(Go 互操作)', () {
      final String publicPem = pubFile.readAsStringSync();
      final String privatePem = privFile.readAsStringSync();
      final RsaEncryptor encryptor = RsaEncryptor();

      const String password = 'secret123';
      const int serverTs = 1754524800000;
      final String ciphertext = encryptor.encryptPassword(
        publicKeyPem: publicPem,
        password: password,
        serverTs: serverTs,
      );

      final Uint8List bytes = base64.decode(ciphertext);
      expect(bytes.length, 256, reason: '2048 位密钥密文应为 256 字节');

      // 裸 RSA 解密。
      final RSAPrivateKey privateKey = _parsePrivateKeyPem(privatePem);
      final BigInt c = _os2ip(bytes);
      final BigInt m = c.modPow(privateKey.exponent!, privateKey.modulus!);
      final Uint8List em = _i2osp(m, 256);

      // EM = 0x00 || maskedSeed(32) || maskedDB(223)。
      expect(em[0], 0, reason: 'EM 前导 0x00(Go RFC 8017 兼容关键)');

      // 复算掩码还原明文:RFC 8017 EME-OAEP 解码。
      final Uint8List maskedSeed = Uint8List.sublistView(em, 1, 33);
      final Uint8List maskedDb = Uint8List.sublistView(em, 33, 256);

      final Uint8List seedMask = _mgf1(maskedDb, 32);
      final Uint8List seed = Uint8List(32);
      for (int i = 0; i < 32; i++) {
        seed[i] = maskedSeed[i] ^ seedMask[i];
      }

      final Uint8List dbMask = _mgf1(seed, 223);
      final Uint8List db = Uint8List(223);
      for (int i = 0; i < 223; i++) {
        db[i] = maskedDb[i] ^ dbMask[i];
      }

      // DB = lHash(32) || PS(全 0) || 0x01 || M。
      final Uint8List lHash = _sha256(Uint8List(0));
      for (int i = 0; i < 32; i++) {
        expect(db[i], lHash[i], reason: 'lHash 应等于 SHA-256("")');
      }
      // 找到 0x01 分隔符。
      int sep = -1;
      for (int i = 32; i < 223; i++) {
        if (db[i] == 0x01) {
          sep = i;
          break;
        }
        expect(db[i], 0, reason: 'PS 应全 0');
      }
      expect(sep, greaterThan(32), reason: '应存在 0x01 分隔符');

      final String plain = utf8.decode(Uint8List.sublistView(db, sep + 1));
      final Map<String, dynamic> payload =
          jsonDecode(plain) as Map<String, dynamic>;
      expect(payload['password'], password);
      expect(payload['ts'], serverTs);
    });

    test('同一明文两次加密产生不同密文(随机 seed)', () {
      final String publicPem = pubFile.readAsStringSync();
      final RsaEncryptor encryptor = RsaEncryptor();
      final String a = encryptor.encryptPassword(
        publicKeyPem: publicPem,
        password: 'pwd',
        serverTs: 1754524800000,
      );
      final String b = encryptor.encryptPassword(
        publicKeyPem: publicPem,
        password: 'pwd',
        serverTs: 1754524800000,
      );
      expect(a, isNot(equals(b)));
    });
  });
}

/// 解析 PKCS#8 PEM 私钥 → pointycastle RSAPrivateKey(测试专用)。
///
/// PrivateKeyInfo ::= SEQUENCE { version, algorithm, privateKey OCTET STRING }
/// privateKey 内容 = RSAPrivateKey ::= SEQUENCE { version, n, e, d, p, q, ... }。
RSAPrivateKey _parsePrivateKeyPem(String pem) {
  final String body = pem
      .replaceAll('-----BEGIN PRIVATE KEY-----', '')
      .replaceAll('-----END PRIVATE KEY-----', '')
      .replaceAll(RegExp(r'\s'), '');
  final Uint8List der = base64.decode(body);

  final ASN1Sequence pkcs8 = ASN1Sequence.fromBytes(der);
  final ASN1OctetString octets = pkcs8.elements![2] as ASN1OctetString;
  final ASN1Sequence rsa = ASN1Sequence.fromBytes(octets.octets!);
  final List<ASN1Integer> ints =
      rsa.elements!.whereType<ASN1Integer>().toList();
  // ints = [version, n, e, d, p, q, ...]。
  // RSAPrivateKey(modulus, privateExponent, p, q) → (n, d, p, q)。
  return RSAPrivateKey(
    ints[1].integer!,
    ints[3].integer!,
    ints[4].integer!,
    ints[5].integer!,
  );
}

BigInt _os2ip(Uint8List bytes) {
  BigInt x = BigInt.zero;
  for (final int b in bytes) {
    x = (x << 8) | BigInt.from(b);
  }
  return x;
}

Uint8List _i2osp(BigInt x, int length) {
  final Uint8List out = Uint8List(length);
  BigInt v = x;
  for (int i = length - 1; i >= 0; i--) {
    out[i] = (v & BigInt.from(0xFF)).toInt();
    v = v >> 8;
  }
  return out;
}

Uint8List _sha256(Uint8List input) {
  final SHA256Digest digest = SHA256Digest();
  return digest.process(input);
}

Uint8List _mgf1(Uint8List seed, int maskLen) {
  final BytesBuilder out = BytesBuilder();
  int counter = 0;
  while (out.length < maskLen) {
    final ByteData c = ByteData(4)..setUint32(0, counter);
    final Uint8List input = Uint8List(seed.length + 4)
      ..setRange(0, seed.length, seed)
      ..setRange(seed.length, seed.length + 4, c.buffer.asUint8List());
    out.add(_sha256(input));
    counter++;
  }
  final Uint8List full = out.toBytes();
  return Uint8List.sublistView(full, 0, maskLen);
}
