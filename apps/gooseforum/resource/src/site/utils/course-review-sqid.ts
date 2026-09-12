// 评价短码（sqid）：与 YourTJCourse-Serverless 后端 sqids@0.3.0 同算法
// （自定义去易混字母表 + minLength 4），同一 review id 前台/管理端/serverless 三端短码一致。
// 独立成轻量模块：管理端页面只展示编号，直接引这里，避免从 course-review-share
// 连带加载 html-to-image 导出/头像实现（详见 issue #628 review P2）。

import Sqids from 'sqids'

export const REVIEW_SQID_ALPHABET = 'bcdfghjkmnpqrstvwxyzBCDFGHJKMNPQRSTVWXYZ23456789'

const reviewSqids = new Sqids({ alphabet: REVIEW_SQID_ALPHABET, minLength: 4 })

export function reviewSqid(reviewId: number): string {
  return reviewSqids.encode([reviewId])
}
