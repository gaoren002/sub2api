# PR 文档：OpenAI fast 2.5x 计价

## 标题

```text
fix: apply 2.5x fast pricing for gpt-5.5 and codex-auto-review
```

## 描述

```markdown
## 概要

将 `gpt-5.5` 和 `codex-auto-review` 的 OpenAI fast/priority 计价固定为普通价的 `2.5x`。

## 背景

运行时远程价格表中，`gpt-5.5` 和 `codex-auto-review` 当前 priority 字段仍是普通价 `2x`。这会导致请求携带 `service_tier=priority` 或客户端别名 `fast` 时少计费。

## 修改内容

- 在计费层增加模型级 fast 计价策略：
  - `gpt-5.5` priority input/output/cache read = 普通价 `2.5x`
  - `codex-auto-review` priority input/output/cache read = 普通价 `2.5x`
- 策略在动态价格、fallback 价格、渠道覆盖价格之后生效，避免被远程价格表或自定义基础价中的旧 priority 字段覆盖。
- 为 fallback 价格文件补充 `codex-auto-review` 的 priority 字段。
- 增加回归测试覆盖：
  - 远程价格表写成 `2x` 时仍按 `2.5x` 计费
  - 渠道覆盖基础价后 fast 字段重新按 `2.5x` 计算

## 验证

```bash
go test -tags unit ./internal/service -run 'TestCalculateCostWithServiceTier_(Gpt55PriorityUses2Point5xPricing|CodexAutoReviewPriorityUses2Point5xPricing|Gpt54MiniPriorityFallsBackToTierMultiplier)|TestGetModelPricingWithChannel_Fast25xRecalculatedForChannelOverrides|TestDefaultPricingIncludesCodexAutoReview'
```

## 风险

中低。改动仅影响 `service_tier=priority` 的 `gpt-5.5` 和 `codex-auto-review` token 计费；普通 tier、flex、其他模型保持原逻辑。

## 上线说明

本 PR 只准备代码和文档，暂不上线。合并或部署前建议再跑完整 backend unit 测试。

## 回滚

回滚本 PR 即可恢复为远程价格表或通用 `priority=2x` 的计价行为。
```

## 分支

```text
fix/openai-fast-pricing-25x
```
