# 课程资料 · 腾讯云 COS（阶段 1）

阶段 1 只完成服务端 COS 连通与 Key 约定；业务 API、数据库与前后端 UI 在后续阶段实现。

## 环境变量

在 `backend/.env`（已 git 忽略）中配置：

| 变量 | 说明 | 默认 |
|------|------|------|
| `COS_BUCKET` | Bucket 名称（含 APPID 后缀） | `caveclass-resources-1385365202` |
| `COS_REGION` | 地域 | `ap-guangzhou` |
| `COS_SECRET_ID` | 腾讯云 API 密钥 ID | （必填才可调用 API） |
| `COS_SECRET_KEY` | 腾讯云 API 密钥 Key | （必填才可调用 API） |

**禁止**在前端、`VITE_*` 或仓库中存放 Secret。Bucket 保持**私有** ACL。

## 对象 Key 约定

后端生成，不信任客户端任意 Key：

```text
courses/{courseId}/originals/{uuid}.{ext}
```

扩展名白名单与 `internal/platform/cos/keys.go` 一致（pdf、office、zip、常见图片等）。

## 后续阶段：后端代理上传（推荐路径）

不在浏览器暴露长期密钥；与产品方案一致的安全边界：

1. **教师** `POST /api/teacher/courses/:id/resources/upload-intent`（阶段 2）：服务端校验课程管理权限，生成 `object_key` 与 DB `PENDING` 记录。
2. **上传字节**（二选一，阶段 2 选型）：
   - **A. 后端代理**：`PUT` multipart 到 Gin，服务端流式 `PutObject` 到 COS（实现简单，大文件需注意超时与内存；可配合分片）。
   - **B. 短效 STS / 预签名 PUT**：仅允许写入刚生成的 `object_key` 前缀；前端直传 COS，完成后调 `complete`。
3. **完成** `POST .../complete`：服务端 `HeadObject` 校验大小/MIME，状态 → `READY`。
4. **预览/下载**：经鉴权后返回**短效**预签名 URL 或由 Gin **代理流式下载**（便于即时撤销发布；预签名 URL 在 TTL 内仍可能有效）。

阶段 1 的 `cmd/cos-probe` 与集成测试验证的是服务端 Secret 对 COS 的 Put/Head/Get/Delete 闭环。

## 本地探针

```bash
cd backend
# 在 .env 填入 COS_SECRET_ID / COS_SECRET_KEY 后：
go run ./cmd/cos-probe
```

无凭证时命令会退出并说明缺失项，**不会**伪造成功。

## 预览链路与海外 VPN

当前实现（阶段 2）：

| 类型 | 浏览器实际请求 | 说明 |
|------|----------------|------|
| PDF | 鉴权后返回 COS **预签名 URL**（同时签名 `response-content-disposition=inline` 与 `response-content-type=application/pdf`）；前端 **fetch CORS 拉取整文件** → **pdf.js** 渲染 | 桶 CORS 需放行站点 Origin；香港桶勿用 CI doc-preview |
| 图片 | 预签名 URL 直连 COS | |
| Office（pptx/docx） | 预签名 URL + CI 参数 `ci-process=doc-preview`，**iframe** 嵌入万象生成的 HTML | 仍直连 `*.myqcloud.com` |

鉴权：列表/预览元数据走 `/api`（带 JWT）；预签名 URL 在 TTL 内可被持有链接的人 GET，故 TTL 较短。

### 为什么海外 VPN 容易出问题

1. **路由绕路**：全局 VPN 常把流量先送到欧美再回亚洲，访问广州 COS 延迟高、易超时；pdf.js 多段 Range 请求会放大问题。
2. **跨境链路质量**：部分 VPN 对大陆 IP 段限速或间歇丢包，表现为 iframe 空白、PDF 一直转圈。
3. **不是典型 CORS 问题**：预签名 GET 一般可跨域；pdf.js 使用 `withCredentials: false`。若 Bucket 未配 CORS，控制台会出现 CORS 报错（与 VPN 无关，需在 COS 控制台允许前端 Origin）。
4. **Office iframe**：PPTX 预览依赖数据万象 doc-preview，整页在 COS/CI 域名下渲染；VPN 问题与 PDF 类似，**目前无同源代理**（CI 需在 COS 侧处理），只能优化网络或下载本地打开。

### 课堂演示（香港）建议

1. **优先直连校园/本地网络，不开 VPN**。香港到 `ap-guangzhou` 通常延迟可接受（约数十 ms 级，视运营商而定）。
2. 演示前在教室网络下试开一门课的 **PDF + PPTX** 预览，并试一次 **下载**。
3. 确保后端 `.env` 已配 COS，`COS_DOC_PREVIEW_ENABLED=true` 且 Bucket 已开通文档预览（Office）。
4. 前端开发时：Vite 代理 `/api` 到本地 Gin；**只有预览/下载会访问 myqcloud.com**，与 localhost API 无关。

### VPN 用户自助

1. **分流（Split Tunnel）**：让 `localhost` / 内网 / CaveClass 部署域名走直连；或仅对 `*.myqcloud.com`、`*.tencentcos.cn` 直连。
2. 换 **香港或亚太节点** 的 VPN，避免强制经欧美回程。
3. PDF 加载失败时，在预览里点 **「通过服务器加载」**（调用 `mode=proxy`）；仍失败则 **下载** 后本地查看。
4. PPTX 若 iframe 空白：关 VPN 或分流 myqcloud 后刷新；或改为 PDF 资料 / 下载后用 PowerPoint 演示。

### 腾讯云侧可选优化（运维）

- **全球加速（COS 全球加速域名）**：降低跨境访问广州 Bucket 的延迟（控制台开启并改用加速域名签发 URL，需改后端 COS 客户端域名配置）。
- **CDN 自定义域名 + HTTPS**：静态 GET 走 CDN 边缘（需私有 Bucket 回源鉴权或仍用预签名）。
- **备用地域**：新 Bucket 选 `ap-hongkong` 可缩短 HK 演示路径，但**已有对象需迁移**，非即时切换。

### API 摘要

- `GET /api/courses/:courseId/resources/:resourceId/preview` — 返回 `{ kind, url?, message? }`；PDF 的 `url` 为短时 COS 预签名 GET（含 inline disposition 与 `application/pdf` content-type）
