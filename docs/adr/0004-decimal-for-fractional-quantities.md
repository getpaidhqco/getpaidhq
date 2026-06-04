# Fractional quantities use shopspring/decimal; money stays int64 cents

Go has no native fixed-point decimal type. The stdlib offers only `float64` (binary
floating point — lossy for money) and `math/big` (`Rat`/`Float` — exact but verbose),
and an `int64` of scaled units cannot represent the `Decimal(38,9)` columns the usage
tables declare (~38 significant digits exceeds `int64`'s ~19).

Usage quantities (e.g. 41.6667 seat-hours, GB to several decimals) and sub-cent
per-unit rates are fractional and must be exact, so we add
**`github.com/shopspring/decimal`** for those fields. It is the de-facto Go decimal,
maps cleanly to Postgres `numeric` / ClickHouse `Decimal`, and has GORM support.

**Money** totals and amounts stay **`int64` cents**, rounded once — as they are
everywhere in the codebase today. Decimal is used only for fractional *quantities* and
*unit rates*, not for charged amounts.

Trade-off: a new dependency and a second numeric convention, deliberately scoped to
fractional quantities/rates. The codebase has no decimal library today, so this is the
introduction point.
