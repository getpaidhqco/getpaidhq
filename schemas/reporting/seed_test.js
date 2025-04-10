import { PrismaClient } from '@prisma/client';
import { faker } from '@faker-js/faker';

const prisma = new PrismaClient();

async function main() {
    console.log('Seeding database...');

    const startDate = new Date('2025-01-01');
    const endDate = new Date();

    const orgId = 'mollie'; 
    const currency = 'USD';

    // Seed DailyMetric data
    for (let date = new Date(startDate); date <= endDate; date.setDate(date.getDate() + 1)) {

        try {
            await prisma.dailyMetric.create({
                data: {
                    orgId,
                    date: new Date(date),
                    currency,
                    arr: faker.number.int({ min: 1000, max: 100000 }),
                    mrr: faker.number.int({ min: 10, max: 1000 }),
                    pastDueTotal: faker.number.int({ min: 0, max: 500 }),
                    pastDueCount: faker.number.int({ min: 0, max: 50 }),
                    customerCount: faker.number.int({ min: 10, max: 500 }),
                    churnCount: faker.number.int({ min: 0, max: 20 }),
                    churnTotal: faker.number.int({ min: 0, max: 200 }),
                    churnRate: faker.number.float({ min: 0, max: 50 }),
                    aveRevenuePerUser: faker.number.float({ min: 10, max: 100 }),
                    customerLifetimeValue: faker.number.int({ min: 100, max: 10000 }),
                    successfulPayments: faker.number.int({ min: 0, max: 50 }),
                    failedPayments: faker.number.int({ min: 0, max: 10 }),
                    refundCount: faker.number.int({ min: 0, max: 5 }),
                    refundTotal: faker.number.int({ min: 0, max: 500 }),
                },
            });
        } catch (e) {
            if (e.code === 'P2002') {
                console.warn('Unique constraint violation, ignoring:', e.meta.target);
            } else {
                throw e;
            }
        }
    }

    // Seed Customers data
    const customers = [];
    for (let i = 0; i < 50; i++) {
        const customerId = faker.string.uuid();
        customers.push(
            await prisma.customers.create({
                data: {
                    orgId,
                    id: customerId,
                },
            })
        );
    }

    // Seed Subscriptions data
    for (let i = 0; i < 50; i++) {
        await prisma.subscription.create({
            data: {
                orgId,
                pspId: faker.string.uuid(),
                status: faker.helpers.arrayElement([
                    'trial',
                    'active',
                    'past_due',
                    'cancelled',
                ]),
                orderId: faker.string.uuid(),
                customerId: faker.helpers.arrayElement(customers).id,
                startDate: faker.date.between({from:startDate, to:endDate}),
                endDate: faker.date.future({refDate:startDate}),
                billingInterval: faker.helpers.arrayElement([
                    'none',
                    'minute',
                    'hour',
                    'day',
                    'week',
                    'month',
                    'year',
                ]),
                billingIntervalQty: faker.number.int({ min: 1, max: 12 }),
                cycles: faker.number.int({ min: 1, max: 24 }),
                billingAnchor: faker.number.int({ min: 1, max: 31 }),
                currency,
                amount: faker.number.int({ min: 5, max: 500 }),
                cyclesProcessed: faker.number.int({ min: 0, max: 10 }),
                totalRevenue: faker.number.int({ min: 0, max: 10000 }),
            },
        });
    }

    // Seed Payments data
    for (let i = 0; i < 100; i++) {
        await prisma.payment.create({
            data: {
                orgId,
                psp: "Paystack",
                recurring: true,
                orderId: faker.string.uuid(),
                amount: faker.number.int({ min: 5, max: 500 }),
                currency,
                status: faker.helpers.arrayElement([
                    'pending',
                    'failed',
                    'succeeded',
                    'refunded',
                    'partial_refund',
                    'cancelled',
                    'expired',
                    'fraudulent',
                ]),
                psp_fee: faker.number.int({ min: 0, max: 50 }),
                platform_fee: faker.number.int({ min: 0, max: 20 }),
                net_amount: faker.number.int({ min: 0, max: 300 }),
                completedAt: faker.date.between({from:startDate, to:endDate}),
            },
        });
    }

    // Seed Refund data
    for (let i = 0; i < 20; i++) {
        await prisma.refund.create({
            data: {
                orgId,
                paymentId: faker.string.uuid(),
                currency,
                amount: faker.number.int({ min: 1, max: 500 }),
                date: faker.date.between({from:startDate, to:endDate}),
                reason: faker.lorem.sentence(),
                usd_amount: faker.number.int({ min: 1, max: 500 }),
            },
        });
    }

    console.log('Database seeding completed!');
}

main()
    .catch((e) => {
        console.error(e);
        process.exit(1);
    })
    .finally(async () => {
        await prisma.$disconnect();
    });