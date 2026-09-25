# Content rating, audience and app access

## IARC content rating questionnaire — expected answers

POS.Go is a business tool with no user-to-user communication, no gambling, no
violence, no mature content and no unrestricted web browsing. The questionnaire
answers should come out as **Everyone / PEGI 3 / ESRB Everyone** with no
questionnaires.

| Question | Answer | Note |
|---|---|---|
| Violence, blood, weapons | No | |
| Sexuality or nudity | No | The community directory shows staff names, roles and companies only |
| Language | No | User-entered free text exists in profiles and job posts, but the app ships no profanity; answer "No" and be ready to justify |
| Controlled substances | No | |
| Gambling | No | |
| User-generated content / chat | Partly | Staff profiles and company pages accept free text and are visible inside the tenant. Declare "Users can interact" only if Play's screen asks about it, and moderate via the community moderation endpoints |
| Sharing location | No | |
| Digital purchases | **Yes — see below** | |

### Digital purchases — the one item that needs care

The app ships with **no Google Play Billing integration**: scan credits can be
topped up in-app, but only against the merchant's own backend balance, not
Play. Two defensible positions:

1. **No Play Billing integration → the "in-app purchases" questions in Play's
   content rating / app content pages can be answered No**, provided the top-up
   only ever spends a balance that was granted on the server (a trial allowance),
   not a currency purchase made inside the app. **[verify]**
2. If a real money purchase of credits is reachable from the app, Play Billing
   becomes mandatory and the top-up screen has to move behind it. That is a
   product change, not a listing change.

Confirm with the product owner which of the two applies to 1.0.0 before
submitting.

## Target audience and content

- Target age group: **18 and over** (it is a cashier/POS tool with financial
  records and staff management).
- Appeals to children: No.
- Ads: No. No ad SDK is bundled.
- In-app purchases: see above.
- Generative AI: No. The OCR invoice scanner is a first-party backend service,
  not a user-facing generative-AI feature. **[verify]**
- COVID-19 / health-related: No.
- User accounts: Yes, per merchant store, created by the store owner.
- Data is encrypted in transit, and the app is not a "financial features" app in
  Play's sense (it records sales amounts, it does not move money).

## App access (all functionality unrestricted)

POS.Go has no gated functionality: a reviewer can create a store from the
sign-in screen and reach every feature. Choose **"All functionality is
unrestricted"** and leave the credentials fields empty.

For completeness, if a reviewer cannot receive the confirmation email, seed a
demo store and paste the credentials here before submitting:

```
Store ID:   TBD
Email:      TBD
Password:   TBD
```

## Government apps, financial features, health

- Government app: No.
- Financial features: the app records sales, tenders and refunds. It does not
  lend, trade, or hold funds. If Play's financial-features declaration captures
  "receipt or transaction records", answer accordingly. **[verify]**
- Health: No.
