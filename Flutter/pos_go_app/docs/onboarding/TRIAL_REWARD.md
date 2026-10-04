# Trial Reward

## Objective

Eligible trial users who successfully complete the required onboarding workflow may receive an extension up to 30 total days.

## Server Authority

The client must never determine or grant the reward.

## Eligibility

Example:

```text
account_type == TRIAL
AND onboarding_activation_completed == true
AND reward_not_already_granted == true
```

## Idempotency

Use a unique reward type such as:

`ONBOARDING_30_DAY_EXTENSION`

The reward must be granted at most once.

## UX

```text
🎉 مبروك!

أكملت إعداد متجرك بنجاح.

🎁 أصبحت تجربتك مجانية حتى 30 يومًا.

[ ابدأ البيع ]
```
