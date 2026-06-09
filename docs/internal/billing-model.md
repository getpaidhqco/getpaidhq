


Matching is AND-across-keys
So a filter means "price differs along this dimension."


- Filter = the dimension that sets the rate. Few, fixed categories; different prices.
- Group = a dimension you want broken out on the invoice at the same rate. Many, open-ended identities; a different price per value would make no sense.



The one-line principle

- Filter = the dimension that sets the rate. Few, fixed categories; different prices.
- Group = a dimension you want broken out on the invoice at the same rate. Many, open-ended identities; a different price per value would make no sense.

If you'd never charge two values of a dimension differently, it's a group, not a filter.

A clean example: an SMS/messaging API

You bill messages. The rate depends on message type — SMS is cheap, MMS is dear:

SMS  = $0.01 / message
MMS  = $0.05 / message
That's a filter on type. Two values, genuinely different prices.

Your customer is an agency running messages for many client projects. They want the invoice to show how much each project cost — but every project pays the same per-message rate. There's no such thing as "MMS
costs more for project-X than project-Y." So project is a group, not a filter.

The events in a period

type   project     count
SMS    acme         1000
SMS    globex        500
MMS    acme          200
MMS    initech       100

How it bills (pseudo-code)

for each TYPE bucket (filter — picks the rate):
rate = bucket.rate
for each distinct PROJECT in that bucket (group — splits the line):
units  = count of messages in (this type, this project)
amount = units * rate          # rate is the same for every project
emit one invoice line

Filter chooses the rate (outer loop). Group chooses how the line is split (inner loop). The rate never changes inside the group — only the count does.

The resulting invoice

SMS   project=acme     1000 × $0.01 = $10.00
SMS   project=globex    500 × $0.01 =  $5.00
MMS   project=acme      200 × $0.05 = $10.00
MMS   project=initech   100 × $0.05 =  $5.00
──────
$30.00

Same scenario without the group — just the filters:

SMS   1500 × $0.01 = $15.00
MMS    300 × $0.05 = $15.00
──────
$30.00

Identical total. Same rates. The only difference is line granularity — the group split the SMS line into per-project lines and the MMS line into per-project lines. That's all a group does: it subdivides a
priced line into one line per value, at the same rate.

Why project can't just be a filter

1. New projects appear all the time.
   A filter must list every value up front. An unlisted project
   → matches nothing → falls to the default price (wrong/blank).
   A group discovers projects from the events automatically.

2. You'd be duplicating the same rate across every project cell,
   for no reason — and they'd drift when you change the price.

3. You never want a different price per project. The split is for
   VISIBILITY (cost attribution), not for PRICING.

The decision test

Ask one question about the dimension:

"Would I ever set a different per-unit price for two values of this?"

     YES → it's a FILTER   (type: SMS vs MMS, hot vs cold storage, US vs intl)
     NO  → it's a GROUP    (project, api_key, user_id, environment, phone number)

