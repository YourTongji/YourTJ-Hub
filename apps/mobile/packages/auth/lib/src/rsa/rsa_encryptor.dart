import 'dart:convert';
import 'dart:math';
import 'dart:typed_data';

import 'package:pointycastle/asn1.dart';
import 'package:pointycastle/asymmetric/api.dart';
import 'package:pointycastle/digests/sha256.dart';

/// RSAES-OAEP(SHA-256, MGF1-SHA256)加密,严格对齐 Go `rsa.EncryptOAEP`
///(RFC 8017 7.1.1:EM 带前导 `0x00` 字节)。
///
/// **互操作警告**:pointycastle 自带的 `OAEPEncoding` 实现的是 RFC 2437 v2.0
/// (EM 无前导 `0x00`),与 Go `rsa.EncryptOAEP` 不兼容,直接使用会导致后端
/// 无法解密。因此这里手动实现 EME-OAEP 编码(带 `0x00` 前缀)+ 裸 RSAEP。
///
/// 后端契约(已核实 logincrypto.go / authController.go):
/// - 公钥来自 `GET /api/login-public-key`,PEM 编码的 SPKI `publicKey` 字段;
/// - 明文 = `{"password": "...", "ts": <serverTs 毫秒>}` JSON;
/// - 加密 = RSA-OAEP(SHA-256, MGF1-SHA256),输出 base64(标准,非 URL-safe);
/// - payload 3 分钟有效;
/// - 密钥固定 2048 位(`keyBits = 2048`)。
class RsaEncryptor {
  /// 2048 位 RSA 密钥的字节长度。
  static const int _keyLength = 256;

  /// SHA-256 摘要字节长度。
  static const int _hashLength = 32;

  /// 加密 `{"password": pwd, "ts": serverTs}`,返回 base64 密文。
  String encryptPassword({
    required String publicKeyPem,
    required String password,
    required int serverTs,
  }) {
    final RSAPublicKey publicKey = _parsePublicKeyPem(publicKeyPem);
    final String plain = jsonEncode({'password': password, 'ts': serverTs});
    final Uint8List message = Uint8List.fromList(utf8.encode(plain));
    final Uint8List em = _encodeOaep(message);
    final Uint8List ciphertext = _rsaep(publicKey, em);
    return base64.encode(ciphertext);
  }

  /// RFC 8017 7.1.1 EME-OAEP 编码(带前导 `0x00`)。
  Uint8List _encodeOaep(Uint8List message) {
    final int messageLength = message.length;
    final int maxMessageLength = _keyLength - 2 * _hashLength - 2;
    if (messageLength > maxMessageLength) {
      throw ArgumentError(
          'message too long for RSA-OAEP ($messageLength > $maxMessageLength)');
    }

    // lHash = Hash(L),L 为空字符串 → SHA-256("")。
    final Uint8List lHash = SHA256Digest().process(Uint8List(0));

    // DB = lHash || PS || 0x01 || M,PS 全 0。
    final int psLength = _keyLength - messageLength - 2 * _hashLength - 2;
    final Uint8List db = Uint8List(_hashLength + psLength + 1 + messageLength);
    db.setRange(0, _hashLength, lHash);
    db.setRange(_hashLength + psLength + 1, db.length, message);
    db[_hashLength + psLength] = 0x01;

    // seed 随机;_k - hLen - 1 = 223 字节的 DB 掩码。
    final Uint8List seed = _randomBytes(_hashLength);
    final Uint8List dbMask = _mgf1(seed, _keyLength - _hashLength - 1);
    final Uint8List maskedDb = Uint8List(db.length);
    for (int i = 0; i < db.length; i++) {
      maskedDb[i] = db[i] ^ dbMask[i];
    }

    final Uint8List seedMask = _mgf1(maskedDb, _hashLength);
    final Uint8List maskedSeed = Uint8List(_hashLength);
    for (int i = 0; i < _hashLength; i++) {
      maskedSeed[i] = seed[i] ^ seedMask[i];
    }

    // EM = 0x00 || maskedSeed || maskedDB(前导 0x00 是 Go 兼容的关键)。
    return Uint8List(_keyLength)
      ..setRange(1, 1 + _hashLength, maskedSeed)
      ..setRange(1 + _hashLength, _keyLength, maskedDb);
  }

  /// MGF1(SHA-256),RFC 8017 B.2.1。
  Uint8List _mgf1(Uint8List seed, int maskLen) {
    final SHA256Digest digest = SHA256Digest();
    final BytesBuilder out = BytesBuilder();
    int counter = 0;
    while (out.length < maskLen) {
      final ByteData c = ByteData(4)..setUint32(0, counter);
      final Uint8List input = Uint8List(seed.length + 4)
        ..setRange(0, seed.length, seed)
        ..setRange(seed.length, seed.length + 4, c.buffer.asUint8List());
      out.add(digest.process(input));
      counter++;
    }
    final Uint8List full = out.toBytes();
    return Uint8List.sublistView(full, 0, maskLen);
  }

  /// RSAEP:m = OS2IP(EM);c = m^e mod n;输出 I2OSP(c, k)。
  Uint8List _rsaep(RSAPublicKey key, Uint8List em) {
    final BigInt m = _os2ip(em);
    final BigInt c = m.modPow(key.exponent!, key.modulus!);
    return _i2osp(c, _keyLength);
  }

  Uint8List _randomBytes(int length) {
    final Random rng = Random.secure();
    return Uint8List.fromList(List.generate(length, (_) => rng.nextInt(256)));
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

  /// 解析 PEM 编码的 SPKI 公钥 → pointycastle RSAPublicKey。
  RSAPublicKey _parsePublicKeyPem(String pem) {
    final String body = pem
        .replaceAll('-----BEGIN PUBLIC KEY-----', '')
        .replaceAll('-----END PUBLIC KEY-----', '')
        .replaceAll(RegExp(r'\s'), '');
    final Uint8List der = base64.decode(body);

    // SubjectPublicKeyInfo ::= SEQUENCE { algorithm SEQUENCE { OID, NULL }, BIT STRING }
    final ASN1Sequence spki = ASN1Sequence.fromBytes(der);
    final ASN1BitString bitString = spki.elements![1] as ASN1BitString;
    // BIT STRING 内容 = RSAPublicKey ::= SEQUENCE { INTEGER n, INTEGER e }
    final Uint8List rsaDer = Uint8List.fromList(bitString.stringValues!);
    final ASN1Sequence rsaKey = ASN1Sequence.fromBytes(rsaDer);
    final ASN1Integer n = rsaKey.elements![0] as ASN1Integer;
    final ASN1Integer e = rsaKey.elements![1] as ASN1Integer;
    return RSAPublicKey(n.integer!, e.integer!);
  }
}
