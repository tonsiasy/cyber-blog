# ADR 002: 数据建模引入系列专栏 (Series) 与主题分组 (Category)

## 状态 (Status)
已采纳 (Accepted) - 2026-10-13

## 上下文 (Context)
在 Gin 课程大作业的基线要求中，系统仅需实现 `users`, `posts`, `comments` 三张基础表。
然而，`cyber-blog` 目标是作为长久运营的现代化个人品牌与 Web3 技术独立站。若仅有单向线性的文章列表，会导致长篇系统性技术文章（如 Web3 实战系列、Go 底层原理解析）缺乏目录结构与脉络索引，严重制约读者阅读体验与个人品牌的专业度。

## 决策 (Decision)
我们在数据模型层做出以下扩展：
1. **新增 `Category` (主题分组)**：负责知识维度的宏观分类（如 Web3、系统架构、AI）。
2. **新增 `Series` (系列专栏)**：负责具有递进学习路线的连载专题，支持独立的封面、描述与作者归属。
3. **扩展 `Post` (文章实体)**：
   - 增加外键 `category_id` 与 `series_id`（均设为可选/NULLABLE）。
   - 增加 `order_index` 字段，用于在专栏内进行显式章节排序。
   - 增加 `slug`、`summary`、`cover_image`（对接 Cloudflare R2）。

## 结果与权衡 (Consequences)

### 正向影响 (Positive)
- **100% 向后兼容课程作业**：所有新字段与关联均为可选，标准作业评测逻辑无感知，无需任何破坏性修改。
- **大幅提升前端体验与 SEO**：原生支持按专栏连载、章节跳转和分类聚合，Clean URL (Slug) 极大改善搜索引擎友好度。
- **架构具备演进弹性**：专栏和主题建立后，未来可无缝引入阅读进度追踪、系列专栏 NFT 铸造等 Web3 扩展玩法。

### 潜在限制 (Negative)
- 数据库连接与查询时，涉及的联表预加载（Preload）开销略有增加。
- *缓解措施*：在 `posts(series_id, order_index)` 和 `posts(category_id)` 上建立针对性复合索引，避免全表扫描。
