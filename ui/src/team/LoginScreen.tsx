import { FormEvent, useState } from 'react';
import { useTeam } from './context';
import { Icon } from './icons';
import { Button } from './primitives';

export function LoginScreen() {
  const { login } = useTeam();
  const [gatewayToken, setGatewayToken] = useState('');
  const [userToken, setUserToken] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    if (!userToken.trim()) {
      setError('请输入个人 Team Token');
      return;
    }
    setBusy(true);
    setError('');
    try {
      await login({ gatewayToken, userToken });
      setGatewayToken('');
      setUserToken('');
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason));
    } finally {
      setBusy(false);
    }
  };

  return (
    <main className="login-layout">
      <section className="login-copy">
        <div className="brand-lockup">
          <span className="brand-mark">G</span>
          <div><strong>GoClaw</strong><span>Team Console</span></div>
        </div>
        <h1>团队研发控制面</h1>
        <p>统一需求澄清、审批、开发任务、Runner、证据和项目知识，同时保持每位成员的 Codex OAuth 留在自己的电脑。</p>
        <ul>
          <li><Icon name="shield" />个人身份与项目级 RBAC</li>
          <li><Icon name="branch" />任务、代码、Bug 与证据完整关联</li>
          <li><Icon name="document" />Markdown 与 Catalog 生命周期治理</li>
        </ul>
      </section>
      <form className="login-panel" onSubmit={(event) => void submit(event)}>
        <div>
          <h2>登录 Team Console</h2>
          <p>凭据仅用于建立短期安全会话，不会写入浏览器存储。</p>
        </div>
        <label>
          <span>Gateway Token</span>
          <input
            type="password"
            autoComplete="off"
            value={gatewayToken}
            onChange={(event) => setGatewayToken(event.target.value)}
            placeholder="连接边界 Token；本地未启用时可留空"
          />
        </label>
        <label>
          <span>个人 Team Token</span>
          <input
            type="password"
            autoComplete="off"
            value={userToken}
            onChange={(event) => setUserToken(event.target.value)}
            placeholder="每位成员独立使用"
            required
          />
        </label>
        {error ? <div className="form-error"><Icon name="warning" />{error}</div> : null}
        <Button type="submit" tone="accent" busy={busy}>建立安全会话</Button>
        <small>生产环境必须通过 HTTPS、VPN 或 SSH 隧道访问。</small>
      </form>
    </main>
  );
}
