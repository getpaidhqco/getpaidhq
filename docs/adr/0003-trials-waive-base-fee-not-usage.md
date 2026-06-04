# Trials waive the base fee, not usage

A subscription in trial does not pay its flat/base fee, but metered **usage accrued
during the trial IS billed** — on the invoice that runs at trial end. Free-usage trials
are an abuse vector (sign up, burn usage, churn), so "usage billed during trial" is the
safe **default**.

A `Price` may opt into a **fully-free trial** (base *and* usage) for merchants who want
that. Giving trial users a *bounded* free allowance (N free units) is better served by a
future credits/wallet feature and is deferred — not modelled now.

Mechanically: the trial-end invoice omits/zeroes the base subscription line but includes
usage lines for usage in the trial window.

This matches both reference systems: Lago's trial gate applies only to the
subscription-fee line, never to charge (usage) fees; Polar waives the upfront base order
but bills metered usage at the after-trial cycle.
