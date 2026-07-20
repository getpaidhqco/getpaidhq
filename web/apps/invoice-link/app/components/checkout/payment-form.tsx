import { useRef, useState } from "react";
import { CircleAlert, CircleCheck, Loader2, Lock } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "~/lib/utils";
import {
  cardNumberLength,
  cvcLength,
  detectBrand,
  digitsOnly,
  expiryValid,
  formatCardNumber,
  formatExpiry,
  luhnValid,
} from "~/lib/card";
import {
  type CheckoutSession,
  type CheckoutTotals,
  formatMoney,
} from "~/lib/checkout";
import {
  AmexMark,
  AppleLogo,
  CardBrandMark,
  GoogleLogo,
  MastercardMark,
  VisaMark,
} from "./brand-marks";

const COUNTRIES = [
  { code: "US", name: "United States" },
  { code: "GB", name: "United Kingdom" },
  { code: "CA", name: "Canada" },
  { code: "AU", name: "Australia" },
  { code: "DE", name: "Germany" },
  { code: "FR", name: "France" },
  { code: "NL", name: "Netherlands" },
  { code: "ZA", name: "South Africa" },
];

const POSTAL_REQUIRED = new Set(["US", "GB", "CA"]);

type FormValues = {
  email: string;
  card: string;
  expiry: string;
  cvc: string;
  name: string;
  country: string;
  postal: string;
  line1: string;
  city: string;
};

type FieldName = keyof FormValues;

const EMPTY_VALUES: FormValues = {
  email: "",
  card: "",
  expiry: "",
  cvc: "",
  name: "",
  country: "US",
  postal: "",
  line1: "",
  city: "",
};

function validate(
  values: FormValues,
  manualAddress: boolean,
): Partial<Record<FieldName, string>> {
  const errors: Partial<Record<FieldName, string>> = {};
  const brand = detectBrand(digitsOnly(values.card));
  const cardDigits = digitsOnly(values.card);

  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(values.email)) {
    errors.email = "Enter a valid email address.";
  }
  if (
    cardDigits.length !== cardNumberLength(brand) ||
    !luhnValid(cardDigits)
  ) {
    errors.card = "Enter a valid card number.";
  }
  if (!expiryValid(values.expiry)) {
    errors.expiry = "Enter a valid expiry date.";
  }
  if (digitsOnly(values.cvc).length !== cvcLength(brand)) {
    errors.cvc = "Enter a valid security code.";
  }
  if (!values.name.trim()) {
    errors.name = "Enter the name on your card.";
  }
  if (POSTAL_REQUIRED.has(values.country) && !values.postal.trim()) {
    errors.postal =
      values.country === "US" ? "Enter a ZIP code." : "Enter a postal code.";
  }
  if (manualAddress) {
    if (!values.line1.trim()) errors.line1 = "Enter an address.";
    if (!values.city.trim()) errors.city = "Enter a city.";
  }
  return errors;
}

function FieldError({ id, message }: { id: string; message?: string }) {
  if (!message) return null;
  return (
    <p id={id} className="text-sm text-destructive">
      {message}
    </p>
  );
}

const SELECT_CLASSES =
  "col-span-full row-start-1 h-11 w-full appearance-none rounded-md border border-input bg-transparent pr-8 pl-3 text-base text-foreground shadow-xs outline-none transition-[color,box-shadow] focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 sm:h-10 md:text-sm";

function CountrySelect({
  value,
  onChange,
}: {
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <span className="inline-grid w-full grid-cols-[1fr_--spacing(8)]">
      <select
        name="country"
        aria-label="Country"
        autoComplete="billing country"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className={SELECT_CLASSES}
      >
        {COUNTRIES.map((country) => (
          <option key={country.code} value={country.code}>
            {country.name}
          </option>
        ))}
      </select>
      <svg
        viewBox="0 0 8 5"
        width="8"
        height="5"
        fill="none"
        className="pointer-events-none col-start-2 row-start-1 place-self-center"
      >
        <path d="M.5.5 4 4 7.5.5" stroke="currentcolor" />
      </svg>
    </span>
  );
}

type PaymentStatus = "idle" | "processing" | "succeeded" | "declined";

export function PaymentForm({
  session,
  totals,
  initialStatus = "idle",
}: {
  session: CheckoutSession;
  totals: CheckoutTotals;
  initialStatus?: PaymentStatus;
}) {
  const [status, setStatus] = useState<PaymentStatus>(initialStatus);
  const [values, setValues] = useState<FormValues>(EMPTY_VALUES);
  const [touched, setTouched] = useState<Partial<Record<FieldName, boolean>>>(
    {},
  );
  const [manualAddress, setManualAddress] = useState(false);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const cardDigits = digitsOnly(values.card);
  const brand = detectBrand(cardDigits);
  const errors = validate(values, manualAddress);
  const formValid = Object.keys(errors).length === 0;
  const processing = status === "processing";

  const setValue = (field: FieldName, value: string) =>
    setValues((prev) => ({ ...prev, [field]: value }));
  const touch = (field: FieldName) =>
    setTouched((prev) => ({ ...prev, [field]: true }));
  const errorFor = (field: FieldName) =>
    touched[field] ? errors[field] : undefined;

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!formValid || processing) return;
    setStatus("processing");
    // Demo gateway: 4000 0000 0000 0002 simulates a decline.
    timeoutRef.current = setTimeout(() => {
      setStatus(cardDigits.endsWith("0002") ? "declined" : "succeeded");
    }, 1500);
  }

  if (status === "succeeded") {
    return (
      <div className="flex flex-col items-start gap-5">
        <CircleCheck className="size-6 shrink-0 text-emerald-600" />
        <div>
          <h2 className="text-lg font-semibold text-balance text-foreground">
            Payment successful
          </h2>
          <p className="mt-1 text-base/7 text-pretty text-muted-foreground sm:text-sm/6">
            Your subscription is active. We've sent a receipt to{" "}
            {values.email || "your email address"}.
          </p>
        </div>
        <Button asChild size="lg" className="w-full">
          <a href={session.merchant.returnUrl}>
            Return to {session.merchant.name}
          </a>
        </Button>
      </div>
    );
  }

  return (
    <form
      onSubmit={handleSubmit}
      noValidate
      className="flex flex-col gap-6"
      aria-busy={processing}
    >
      <div className="flex flex-col gap-3">
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <button
            type="button"
            aria-label={`Pay ${formatMoney(totals.total, session)} with Apple Pay`}
            disabled={processing}
            className="flex h-11 items-center justify-center gap-1 rounded-md bg-black text-white hover:bg-black/85 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring disabled:opacity-50 sm:h-10"
          >
            <AppleLogo />
            <span className="text-base font-medium sm:text-sm">Pay</span>
          </button>
          <button
            type="button"
            aria-label={`Pay ${formatMoney(totals.total, session)} with Google Pay`}
            disabled={processing}
            className="flex h-11 items-center justify-center gap-1.5 rounded-md bg-white text-foreground ring-1 ring-black/10 hover:bg-black/2.5 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring disabled:opacity-50 sm:h-10"
          >
            <GoogleLogo />
            <span className="text-base font-medium sm:text-sm">Pay</span>
          </button>
        </div>
        <div className="flex items-center gap-4">
          <span className="h-px flex-1 bg-black/10" />
          <p className="text-sm text-muted-foreground">Or pay with card</p>
          <span className="h-px flex-1 bg-black/10" />
        </div>
      </div>

      <div className="flex flex-col gap-1.5">
        <h2 className="text-base font-medium text-foreground sm:text-sm">
          Contact
        </h2>
        <Input
          type="email"
          name="email"
          aria-label="Email address"
          placeholder="Email address"
          autoComplete="email"
          value={values.email}
          aria-invalid={!!errorFor("email") || undefined}
          aria-describedby={errorFor("email") ? "email-error" : undefined}
          onChange={(event) => setValue("email", event.target.value)}
          onBlur={() => touch("email")}
          disabled={processing}
          className="h-11 sm:h-10"
        />
        <FieldError id="email-error" message={errorFor("email")} />
      </div>

      <div className="flex flex-col gap-1.5">
        <h2 className="text-base font-medium text-foreground sm:text-sm">
          Payment method
        </h2>
        <div className="relative">
          <Input
            name="card-number"
            aria-label="Card number"
            placeholder="Card number"
            autoComplete="cc-number"
            inputMode="numeric"
            value={values.card}
            aria-invalid={!!errorFor("card") || undefined}
            aria-describedby={errorFor("card") ? "card-error" : undefined}
            onChange={(event) => {
              const digits = digitsOnly(event.target.value).slice(
                0,
                cardNumberLength(detectBrand(digitsOnly(event.target.value))),
              );
              setValue("card", formatCardNumber(digits, detectBrand(digits)));
            }}
            onBlur={() => touch("card")}
            disabled={processing}
            className="h-11 pr-24 sm:h-10"
          />
          <span className="pointer-events-none absolute top-1/2 right-3 flex -translate-y-1/2 gap-1">
            {brand === "unknown" ? (
              <>
                <VisaMark />
                <MastercardMark />
                <AmexMark />
              </>
            ) : (
              <CardBrandMark brand={brand} />
            )}
          </span>
        </div>
        <FieldError id="card-error" message={errorFor("card")} />
        <div className="grid grid-cols-2 gap-3">
          <div className="flex flex-col gap-1.5">
            <Input
              name="card-expiry"
              aria-label="Expiry date"
              placeholder="MM / YY"
              autoComplete="cc-exp"
              inputMode="numeric"
              value={values.expiry}
              aria-invalid={!!errorFor("expiry") || undefined}
              aria-describedby={
                errorFor("expiry") ? "expiry-error" : undefined
              }
              onChange={(event) =>
                setValue("expiry", formatExpiry(event.target.value))
              }
              onBlur={() => touch("expiry")}
              disabled={processing}
              className="h-11 sm:h-10"
            />
            <FieldError id="expiry-error" message={errorFor("expiry")} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Input
              name="card-cvc"
              aria-label="Security code"
              placeholder="CVC"
              autoComplete="cc-csc"
              inputMode="numeric"
              value={values.cvc}
              aria-invalid={!!errorFor("cvc") || undefined}
              aria-describedby={errorFor("cvc") ? "cvc-error" : undefined}
              onChange={(event) =>
                setValue(
                  "cvc",
                  digitsOnly(event.target.value).slice(0, cvcLength(brand)),
                )
              }
              onBlur={() => touch("cvc")}
              disabled={processing}
              className="h-11 sm:h-10"
            />
            <FieldError id="cvc-error" message={errorFor("cvc")} />
          </div>
        </div>
        <Input
          name="card-name"
          aria-label="Name on card"
          placeholder="Name on card"
          autoComplete="cc-name"
          value={values.name}
          aria-invalid={!!errorFor("name") || undefined}
          aria-describedby={errorFor("name") ? "name-error" : undefined}
          onChange={(event) => setValue("name", event.target.value)}
          onBlur={() => touch("name")}
          disabled={processing}
          className="h-11 sm:h-10"
        />
        <FieldError id="name-error" message={errorFor("name")} />
      </div>

      <div className="flex flex-col gap-1.5">
        <h2 className="text-base font-medium text-foreground sm:text-sm">
          Billing address
        </h2>
        <div className="grid grid-cols-2 gap-3">
          <CountrySelect
            value={values.country}
            onChange={(value) => setValue("country", value)}
          />
          <div className="flex flex-col gap-1.5">
            <Input
              name="postal-code"
              aria-label={values.country === "US" ? "ZIP code" : "Postal code"}
              placeholder={values.country === "US" ? "ZIP" : "Postal code"}
              autoComplete="billing postal-code"
              value={values.postal}
              aria-invalid={!!errorFor("postal") || undefined}
              aria-describedby={
                errorFor("postal") ? "postal-error" : undefined
              }
              onChange={(event) => setValue("postal", event.target.value)}
              onBlur={() => touch("postal")}
              disabled={processing}
              className="h-11 sm:h-10"
            />
            <FieldError id="postal-error" message={errorFor("postal")} />
          </div>
        </div>
        {manualAddress ? (
          <>
            <Input
              name="address-line1"
              aria-label="Address"
              placeholder="Address"
              autoComplete="billing address-line1"
              value={values.line1}
              aria-invalid={!!errorFor("line1") || undefined}
              aria-describedby={errorFor("line1") ? "line1-error" : undefined}
              onChange={(event) => setValue("line1", event.target.value)}
              onBlur={() => touch("line1")}
              disabled={processing}
              className="h-11 sm:h-10"
            />
            <FieldError id="line1-error" message={errorFor("line1")} />
            <Input
              name="address-city"
              aria-label="City"
              placeholder="City"
              autoComplete="billing address-level2"
              value={values.city}
              aria-invalid={!!errorFor("city") || undefined}
              aria-describedby={errorFor("city") ? "city-error" : undefined}
              onChange={(event) => setValue("city", event.target.value)}
              onBlur={() => touch("city")}
              disabled={processing}
              className="h-11 sm:h-10"
            />
            <FieldError id="city-error" message={errorFor("city")} />
          </>
        ) : (
          <div>
            <button
              type="button"
              onClick={() => setManualAddress(true)}
              className="text-base font-medium text-primary hover:text-primary/80 sm:text-sm"
            >
              Enter address manually
            </button>
          </div>
        )}
      </div>

      <div className="flex flex-col gap-3">
        {status === "declined" && (
          <div
            role="alert"
            className="flex items-start gap-2 rounded-md bg-destructive/10 p-3"
          >
            <span className="flex h-lh items-center text-sm">
              <CircleAlert className="size-4 shrink-0 text-destructive" />
            </span>
            <p className="text-sm text-pretty text-destructive">
              Your card was declined. Try a different card or contact your
              bank.
            </p>
          </div>
        )}
        <Button
          type="submit"
          size="lg"
          disabled={!formValid || processing}
          className="w-full"
        >
          {processing ? (
            <>
              <Loader2 className="size-4 shrink-0 animate-spin" />
              Processing…
            </>
          ) : (
            `Pay ${formatMoney(totals.total, session)}`
          )}
        </Button>
        <p className="text-sm text-pretty text-muted-foreground">
          By subscribing, you authorize {session.merchant.name} to charge you{" "}
          {formatMoney(totals.recurring, session)} plus metered usage every{" "}
          {session.interval} until you cancel.
        </p>
        <p className="flex items-center justify-center gap-1.5 text-sm text-muted-foreground">
          <Lock className="size-4 shrink-0" />
          Payments are encrypted and secure
        </p>
      </div>
    </form>
  );
}
