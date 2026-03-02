# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 1.x     | ✅ Yes    |

## Reporting a Vulnerability

**请勿通过 GitHub Issues 公开报告安全漏洞。**

如果您发现安全漏洞，请通过以下方式联系我们：

1. **GitHub Private Vulnerability Reporting**（推荐）：  
   访问 [Security Advisories](../../security/advisories/new) 提交私密漏洞报告。

2. **Email**：将漏洞详情发送至项目维护者（请在 GitHub Profile 中查找联系方式）。

请在报告中包含：
- 漏洞类型和影响范围
- 复现步骤（PoC 代码）
- 受影响的版本号
- 建议的修复方案（可选）

我们承诺在 **7 个工作日内**对漏洞报告作出响应，并在 **30 天内**发布修复版本（视严重程度而定）。

---

## Security Design

本 SDK 实现了以下安全机制：

| 机制 | 实现 |
|------|------|
| API Key 传输 | HTTP Header（非 URL 参数），强制 HTTPS |
| 调试日志脱敏 | API Key 在日志中显示为 `****xxxx`（末 4 位） |
| Webhook 签名验证 | SHA-256 + `crypto/subtle` 时间恒定比较（防时序攻击） |
| Webhook 防重放 | 时间戳校验，拒绝超过 ±5 分钟的请求 |
| 零外部依赖 | 无第三方依赖，消除供应链攻击风险 |
| 熔断器 | 防止上游故障级联传播 |

---

## Known Limitations

- `WithBaseURL` 选项未对 URL 进行 SSRF 防护校验，请仅在受信任的内部测试环境使用。
- Webhook 防重放依赖事件中的 `timestamp` 字段；如果上游 17Track 推送的事件不含该字段，则不进行时间戳校验（向后兼容）。
