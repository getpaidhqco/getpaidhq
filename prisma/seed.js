const {PrismaClient} = require('@prisma/client');
const {faker} = require('@faker-js/faker');

const prisma = new PrismaClient();

async function seedOrganization(orgId) {
    const now = new Date();

    console.log(`Seeding organization: ${orgId}`);

    // Create organization
    await prisma.org.upsert({
        where: {id: orgId},
        update: {},
        create: {
            id: orgId,
            name: faker.company.name(),
            country: "ZA",
            createdAt: now,
            updatedAt: now,
        },
    });

    // Create gateways
    const gateways = [
        {
            orgId,
            id: 'Paystack',
            name: 'Paystack',
            pspId: 'Paystack',
            active: true,
            createdAt: now,
            updatedAt: now,
        },
        {
            orgId,
            id: 'CheckoutDotCom',
            name: 'CheckoutDotCom',
            pspId: 'CheckoutDotCom',
            active: true,
            createdAt: now,
            updatedAt: now,
        }
    ];

    for (const gateway of gateways) {
        await prisma.gateway.upsert({
            where: {
                orgId_id: {
                    orgId: gateway.orgId,
                    id: gateway.id
                }
            },
            update: {},
            create: gateway,
        });
    }

    // Create settings
    const settings = [
        {
            orgId,
            parentId: 'Paystack',
            id: 'settings',
            value: {
                api_key: "sk_test_e39ce23869e6e677121a5e6ef691a8c3d835f0bb"
            },
            valueType: 'PaystackConfig',
            createdAt: now,
            updatedAt: now,
        },
        {
            orgId,
            parentId: 'CheckoutDotCom',
            id: 'settings',
            value: {
                secret_key: "sk_sbox_g2dxr775jvhnwbvwqbl5qon6kux"
            },
            valueType: 'CheckoutDotComConfig',
            createdAt: now,
            updatedAt: now,
        }
    ];

    for (const setting of settings) {
        await prisma.setting.upsert({
            where: {
                orgId_parentId_id: {
                    orgId: setting.orgId,
                    parentId: setting.parentId,
                    id: setting.id
                }
            },
            update: {},
            create: setting,
        });
    }

    // Create API key
    await prisma.apiKey.upsert({
        where: {
            orgId_id: {
                orgId,
                id: `apikey-${faker.string.alphanumeric(6)}`
            }
        },
        update: {},
        create: {
            orgId,
            id: `apikey-${faker.string.alphanumeric(6)}`,
            key: `sk_${faker.string.alphanumeric(32)}`,
            createdAt: now,
            updatedAt: now,
        },
    });

    // Create cohort
    await prisma.cohort.upsert({
        where: {
            orgId_id: {
                orgId,
                id: 'signup_date'
            }
        },
        update: {},
        create: {
            orgId,
            id: 'signup_date',
            name: 'Signup Date',
            type: 'signup_date',
            metadata: null,
            createdAt: now,
            updatedAt: now,
        },
    });

    // Create a sample customer
    await prisma.customer.upsert({
        where: {
            orgId_id: {
                orgId,
                id: `cus_${faker.string.alphanumeric(8)}`
            }
        },
        update: {},
        create: {
            orgId,
            id: `cus_${faker.string.alphanumeric(8)}`,
            firstName: faker.person.firstName(),
            lastName: faker.person.lastName(),
            email: faker.internet.email(),
            createdAt: now,
            updatedAt: now,
        },
    });

    console.log(`Organization ${orgId} seeded successfully!`);
}

async function main() {
    // Get org ID from command line arguments or use default
    const orgId = process.argv[2] || 'mollie';

    console.log('Start seeding...', orgId);

    try {
        await seedOrganization(orgId);
        console.log('Seeding finished.');
    } catch (error) {
        console.error('Error during seeding:', error);
        throw error;
    }
}

main()
    .catch((e) => {
        console.error(e);
        process.exit(1);
    })
    .finally(async () => {
        await prisma.$disconnect();
    });