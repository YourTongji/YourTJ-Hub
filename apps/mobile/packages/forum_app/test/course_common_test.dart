
import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/pages/courses/course_common.dart';

void main() {
  test('offering meta keeps term, class and teachers for the review card', () {
    final parts = offeringMetaParts(
      const CourseOfferingPayload(
        id: 901,
        termCode: '2025-2026-1',
        className: '17班',
        classCode: '54009917',
        campus: '四平路校区',
        faculty: '马克思主义学院',
        instructors: <String>['王小莉'],
      ),
      locale: 'zh',
      fallbackId: 901,
    );

    // 卡片单行元信息：学期 · 班次（不含班号） · 教师；班号/校区/院系已在
    // 课程头部与开课记录里，不重复。
    expect(parts.cardMeta, '25秋 · 17班 · 王小莉');
    expect(parts.cardMeta, isNot(contains('54009917')));
    expect(parts.cardMeta, isNot(contains('四平路校区')));
    // 分享卡 chip 仍保留「学期 · 班级 · 班号」，不重复教师（教师另有 chip）。
    expect(parts.chip, '25秋 · 17班 · 54009917');
    expect(parts.chip, isNot(contains('王小莉')));
  });

  test('missing offering falls back to the offering id', () {
    final parts = offeringMetaParts(null, locale: 'zh', fallbackId: 77);

    expect(parts.cardMeta, '#77');
    expect(parts.chip, '#77');
  });

  test('offering without class or teachers keeps only the term', () {
    final parts = offeringMetaParts(
      const CourseOfferingPayload(id: 5, termCode: 'custom', campus: '嘉定校区'),
      locale: 'zh',
      fallbackId: 5,
    );

    expect(parts.cardMeta, 'custom');
    expect(parts.chip, 'custom');
  });

  test('rating ring gradient mirrors the web linear gradient', () {
    const Color start = Color(0xFFFE9A00);
    const Color end = Color(0xFF2563EB);

    final LinearGradient gradient = ratingRingGradient(start: start, end: end);

    // 与 web 同一几何：SVG 左下→右上再整体 -90° 旋转 = 右下→左上线性渐变，
    // 没有扫掠渐变的首尾接缝。
    expect(gradient.colors, <Color>[start, end]);
    expect(gradient.begin, Alignment.bottomRight);
    expect(gradient.end, Alignment.topLeft);
  });
}
