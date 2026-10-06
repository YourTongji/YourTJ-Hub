# 信息流运行与恢复

> Doc type: operations guide
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-10-06

## 配置与启用

`Current`：默认模板及dev/main实例均关闭排名、推荐和行为统计；默认灰度20%、原始保留30天。
部署实例的 `feed.ranking_enabled`、`feed.for_you_enabled`、`feed.metrics_enabled`、
`feed.rollout_percent`、`feed.period`、`feed.salt` 经render_config生成对应TOML。
TOML参数在 `ranking`、`ranking.hot`、`ranking.daily`、`feed.for_you`、`feed.metrics`、
`feed.weights`、`feed.alternative_weights`、`feed.experiments` 中定义。
配置热更新先解析与校验整份不可变快照，非法值保留旧快照；完整配置与排名参数分别计算hash。
修改物化权重使新排名立即未就绪，后台重新回填；不同hash的排名不能混用。

`Current`：启用 `ranking.enabled` 后，serve进程中的单worker进行公开时间回填与物化排序，
持久游标每批最多200条；在后台预算内每秒安排一次回填/重建检查，即使排名队列持续非空也会推进，不依赖空闲队列。不会在启动迁移内扫描历史正文。只有公开话题全部匹配当前排名hash才就绪。
`feed-rebuild` 仅请求重建，serve仍为唯一计算者。启用metrics后采集基线，再开启for_you；
入口实验启用且灰度大于0时原始保留必须30天。首次灰度配置为20%，不要把开启开关当作效果验证。

每个环境使用自己的周期标识和salt。周期/完整参数被固定保存，14天后停止纳入，第22天开始
分批账号汇总；参数改变或捕获中断终止周期。恢复后使用新周期标识开始新的观察。

## 资源与故障

`Current`：候选主池最多240、补充池60；元数据读取最多300，最终快照120、每页20。
画像读取三个各50条的流，合并最近100条；快照每账号最多2个，全局最多512/32MiB。
画像4MiB、公共召回1MiB、重复曝光4MiB、旧客户端推断2MiB、浏览合并2MiB、日/分组缓存2MiB，
普通队列8MiB、候选样本队列2MiB。缓存上限是有效负载预算，进程RSS还包括Go运行时与基础服务。
热门、今日热榜及分类公开候选ID共用上述1MiB缓存，按排名hash、来源、分类和数量区分，30秒过期。
召回先按来源限制ID数量，再在至多300条候选元数据上检查首楼可见性，避免在召回排序之前扫描全站楼层。
冷构建全局最多2个、200ms截止，查询顺序执行，未构建成功立即返回有限最新候选。

后台不额外建立连接池：所有新增SQL由一个worker顺序执行，查询默认250ms截止、默认20任务/秒，
每分钟累计工作上限15秒；主池只剩一个连接时让出。排名脏标记与业务变化同事务提交，
版本、随机generation及due时间阻止旧任务确认新变化。被冻结/注销参与者的历史贡献通过同worker
按owner提供的200条批次标记，安静且超过7天的话题不保留周期性计时任务，业务变化会重新唤醒。

原始队列最多50条一批，普通项最多4KiB、保留内存5秒、最多2次重试；候选样本每条最多32KiB。
上报最多50个patch/32KiB；每账号12次/分钟、突发24，全局200次/秒、突发400。
统计关闭即停止接受新行为并丢弃待写队列；已提交的业务事件仍可完成匿名汇总。
停止进程最多尝试2秒排空，失败/丢弃计数可查看；非正常退出使前一周期的数据完整性未知并终止实验。

`feed-report` 输出同一匿名汇总与可用实验区间；`feed-explain TOPIC_ID` 输出当前公开话题的物化分数及规则重算分量，不输出参与者标识；`feed-replay SAMPLE_ID` 回放有限候选样本。新话题群指标 `new_topic_public`、`new_topic_visible_24h`、`new_topic_first_reply` 与 `first_reply_seconds` 按首次公开日记录，可计算首日曝光比例及已获回复群的平均等待；未回复数量与关闭/过期水位不能隐含为零等待。

管理员统计页提供捕获开关、就绪状态、参数、排队字节/条数/最老年龄、丢弃和后台失败、上一epoch
完整性标志。检查匿名指标的served/visible/open/read配对，并按能力版本、参数、默认入口组和权重组
分开比较。分页409、回退、日志丢失不能写成推荐收益。物理清理每小时启动，每表每批最多500条，有积压时每秒继续；后台调度受
上述额度约束；到期先退出读取，清理积压应作为运维故障处理。

## 回滚与备份

`Current`：`feed.for_you.enabled=false` 关闭推荐入口并回退有限最新流；
`feed.for_you.rollout_percent=0` 停止默认入口治疗分配（手动推荐标签仍可用）；
`feed.metrics.enabled=false` 停止行为捕获并终止观察周期；`ranking.enabled=false` 恢复原公开排序。
同时关闭推荐和捕获是完整推荐回滚，保留schema便于恢复，不删除原生点赞/收藏/内容数据。

`Current`：Go内置SQLite备份及部署backup/snapshot/main→dev脚本共用受测试约束的原始表排除清单
`deploy/scripts/feed-raw-tables.json`。SQLite仅清理私有副本后VACUUM并原子发布，源库不被清理；
PostgreSQL dump排除这些表的数据及排名state/schedule。恢复后的旧原始事件、分组、游标均不可用；
匿名汇总和参数记录保留，旧物化分数经过受控重建后才可用于新推荐。自行使用的外部备份也必须采用
同一排除清单，否则不能满足30天原始行为生命周期。已有自定义隐私政策需同步
[默认政策](../../apps/gooseforum/app/models/defaultconfig/pageconfig/app_privacy.md)中的推荐统计条款后启用。
