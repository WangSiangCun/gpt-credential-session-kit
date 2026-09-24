# Credential Session Kit

独立的已有账号凭证获取组件。它只负责一件事：在指定代理下完成已有账号登录，返回 `access_token`、`session_token` 和 `refresh_token`。

## 约束

- 仅已有账号登录，不进入注册流程。
- 所有请求固定使用调用方传入的代理。
- 密码、2FA、Cookie 和 token 只存在于当前进程内，不写入文件、不打印到日志。
- Team、CPA、席位和数据库逻辑不属于本仓库。

## 使用

```python
from credential_session_kit import CredentialSessionClient

client = CredentialSessionClient(
    auth_project=r"C:\\code\\go\\rental-all\\gpt-auto-register",
)
result = client.login(
    email="ACCOUNT_EMAIL",
    password="ACCOUNT_PASSWORD",
    totp_secret="BASE32_TOTP_SECRET",
    proxy_url="socks5://USER:PASSWORD@HOST:PORT",
)
print(result.access_token)
print(result.session_token)
print(result.refresh_token)
```

结果对象也可以通过 `to_dict()` 转成普通字典。失败时抛出 `CredentialSessionError`，只包含稳定错误码，不包含密码、2FA、Cookie、token 或代理认证信息。

## 当前实现

协议实现通过适配器加载现有账号池的 `AuthFlow`，因此账号池和独立组件使用同一套 CSRF、OAuth、指纹、代理固定、密码、TOTP、callback、网页 session 和 RT 流程。后续替换底层实现时，只需保持 `CredentialSessionClient` 接口。
