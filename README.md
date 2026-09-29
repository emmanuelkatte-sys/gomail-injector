# goMail-injector

基于 [wneessen/go-mail](https://github.com/wneessen/go-mail) 的邮件批量注入工具。

## 与 pmta-injector / PHPMailer-injector 的关系

| 层 | 实现 |
|----|------|
| 配置 / CLI / 调度 / 收件人 / 模板 / 头策略 / 反指纹 / Pickup | **与 pmta-injector-clean 对齐**（同一套 `config.yaml` 与命令） |
| SMTP 传输 | **go-mail**（TLS / AUTH / 拨号）；DATA 提交 Builder 产出的原始 RFC822，避免再解析破坏自定义头 |

命令与用法同 pmta-injector：`send` / `validate` / `serve`。

## 构建

```bash
# 本地（需 Go 1.22+）
cd goMail-injector
go mod tidy
go build -o goMail-injector .

# GUI 打包
python gui/assets/repack_gomail_injector.py
```

部署页选择注入器类型 **goMail**，VPS 使用 `hrkdeploy-gomail.sh` 现场 `go mod tidy` + `go build`。

## SMTP 说明

- `smtp.enabled: true` → go-mail 连接 `smtp.host:port`
- 本地 Haraka/PMTA 明文 25：`use_tls: false`（内部 `mail.NoTLS`）
- Pickup：与 pmta 相同，不经过 go-mail
