const { PrismaClient } = require('@prisma/client');
const { faker } = require('@faker-js/faker');

const prisma = new PrismaClient();
const orgId = 'mollie'
const cohorts = [
    {
        orgId,
        id: 'signup_date',
        name: 'Signup Date',
        type: 'signup_date',
        createdAt: faker.date.past(),
        updatedAt: faker.date.past(),
    },
    {
        orgId,
        id: 'geo_country',
        name: 'Country',
        type: 'geo_country',
        createdAt: faker.date.past(),
        updatedAt: faker.date.past(),
    }
]

async function main() {
    console.log('Start seeding...');

    await prisma.org.create({
        data: {
            id: orgId,
            name: 'Mollie',
            country: "ZA",
            createdAt: faker.date.past(),
            updatedAt: faker.date.past(),
        },
    }).catch((e) => {
        if (e.code === 'P2002') {
            console.warn('Conflict detected, ignoring:', e.meta.target);
        } else {
            console.error(e);
            process.exit(1);
        }
    })

    await prisma.customer.create({
        data: {
            orgId: orgId,
            id: "cus_1",
            firstName: faker.person.firstName(),
            lastName: faker.person.lastName(),
            email: faker.internet.email(),
            createdAt: faker.date.past(),
            updatedAt: faker.date.past(),
        },
    }).catch((e) => {
        if (e.code === 'P2002') {
            console.warn('Conflict detected, ignoring:', e.meta.target);
        } else {
            console.error(e);
            process.exit(1);
        }
    })

    await Promise.all([
        prisma.cohort.createMany({
            data: cohorts,
        }).catch((e) => {
            if (e.code === 'P2002') {
                console.warn('Conflict detected, ignoring:', e.meta.target);
            } else {
                console.error(e);
                process.exit(1);
            }
        }),
    ]);

    console.log('Seeding finished.');
}

main()
    .catch((e) => {
        console.error(e);
        process.exit(1);
    })
    .finally(async () => {
        await prisma.$disconnect();
    });