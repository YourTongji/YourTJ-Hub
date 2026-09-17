# 预设表情包资源来源声明（NOTICE）

本目录收录的图片资源来自以下三个开源项目，用于论坛预设表情包（go:embed 嵌入单二进制，
由 seed-stickers 命令导入数据库）。收录日期：**2026-09-15**。清单见同目录 `manifest.json`。

## 1. flowerhd

- 项目：flowerhd（花字表情包）
- 地址：<https://github.com/k4yt3x/flowerhd>
- 许可证：**CC BY 4.0**（Creative Commons Attribution 4.0 International）
- 收录：14 / 15 张 PNG（508K），保留原文件名（含 `.PNG` 大写扩展名）
- 筛选说明：排除单图 >300KB 的 `这河里吗.png`（1.2MB，4000×4000 高清合成图），其余全收。

## 2. EmojiPackage

- 项目：EmojiPackage
- 地址：<https://github.com/getActivity/EmojiPackage>
- 许可证：**Apache License 2.0**（仓库 LICENSE 原文）
- 收录：194 张（4.9M，其中 GIF 动图 30 张），来自 11 个卡通/文字梗类分类目录
  （滑稽 30、回答 35、夸奖 20、文字 25、小鹦鹉 20、程序员 15、表情 12、合体字 5、
  咸鱼 12、鸭子 12、鹦鹉兄弟 8；每目录限量，目录内按文件名序取前 N 张）
- 筛选说明：
  - 只收非真人、卡通/文字梗类分类；**明确剔除**真人影视肖像类分类
    （尔康、罗翔、基佬、撩妹、富婆、身材、干架、装逼、套图、开车、三连等）
    以及截图/预览类目录（picture、QQ、故事、裸辞、红包、文章、反问）和
    单图普遍超限的动图目录（动图）。
  - 单图 >200KB 跳过（共跳过 13 张）。
  - 1 张跨目录同名文件跳过（程序员/666.jpg 与已有文件重名）。

## 3. WXMemeStickers

- 项目：WXMemeStickers（程序员梗微信表情系列）
- 地址：<https://github.com/anzhi0708/WXMemeStickers>
- 许可证：**MIT License**（Copyright (c) 2025 Anji）
- 收录：15 张（1.9M），全部收录（两个系列目录 14 张 PNG + 根目录预览 meme.jpeg）。

## 使用注意

- 所有文件经 `file` 命令验证为真实图片（PNG/JPEG/GIF）。
- `manifest.json` 中 `name` 为表情名（文件名去扩展名），跨包唯一；
  同名冲突时后者加 `-2` 后缀。`pack` 字段标识来源包。
- 再分发需保留并随附本声明及上述各许可证文本（CC BY 4.0 需署名；
  Apache-2.0 需保留许可证声明；MIT 需保留版权与许可声明）。

## 随包许可证

- [flowerhd — CC BY 4.0](LICENSE-flowerhd.txt)，授权声明见[来源 README](https://github.com/k4yt3x/flowerhd#开源许可)。
- [EmojiPackage — Apache License 2.0](LICENSE-EmojiPackage.txt)。
- [WXMemeStickers — MIT License](LICENSE-WXMemeStickers.txt)。
