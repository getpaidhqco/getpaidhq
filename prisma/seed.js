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

    await Promise.all([
        prisma.cohort.createMany({
            data: cohorts,
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