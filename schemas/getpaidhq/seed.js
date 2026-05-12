const { PrismaClient } = require('@prisma/client');

const prisma = new PrismaClient();

// Clerk org + members for the local dev environment.
// Keep these in sync with the Clerk dashboard. The local Org/User PKs ARE the Clerk IDs
// by design — see internal/core/service/org.go:57-60 and internal/adapter/clerk/middleware.go.
const ORG = {
    id: 'org_30u3YjZIXUTJEIi6n0EFKeXh9gK',
    name: 'mollie',
    country: 'ZA',
    timezone: 'Africa/Johannesburg',
    status: 'active',
};

const USERS = [
    {
        id: 'user_3Dc199D2YS51CSF9RKOadTr66qQ',
        email: 'mollie@checkoutjoy.com',
        role: 'admin', // Clerk org:admin → Prisma Role.admin
    },
    {
        id: 'user_30u3RHBC2qqiI3x8ml4kPMdyZj2',
        email: 'mollie+1@checkoutjoy.com',
        role: 'admin',
    },
];

// Mirrors OrgService.Create at internal/core/service/org.go:79-86 + :92.
const API_KEY_ID = 'sk_local_dev_mollie';
const COHORT_ID = 'signup_date';

function ignoreConflict(label) {
    return (e) => {
        if (e.code === 'P2002' || e.code === 'P2025') {
            console.warn(`[${label}] already exists, skipping`);
            return;
        }
        console.error(`[${label}] failed`, e);
        process.exit(1);
    };
}

async function main() {
    console.log(`Seeding org ${ORG.id} (${ORG.name}) and ${USERS.length} member(s)...`);

    await prisma.org.upsert({
        where: { id: ORG.id },
        update: {
            name: ORG.name,
            country: ORG.country,
            timezone: ORG.timezone,
            status: ORG.status,
        },
        create: ORG,
    });

    for (const u of USERS) {
        await prisma.user.upsert({
            where: { id: u.id },
            update: { email: u.email },
            create: { id: u.id, email: u.email },
        }).catch(ignoreConflict(`user:${u.id}`));

        await prisma.userOrg.upsert({
            where: { userId_orgId: { userId: u.id, orgId: ORG.id } },
            update: { role: u.role },
            create: { userId: u.id, orgId: ORG.id, role: u.role },
        }).catch(ignoreConflict(`userOrg:${u.id}`));
    }

    await prisma.apiKey.upsert({
        where: { orgId_id: { orgId: ORG.id, id: API_KEY_ID } },
        update: { key: API_KEY_ID },
        create: { orgId: ORG.id, id: API_KEY_ID, key: API_KEY_ID },
    }).catch(ignoreConflict(`apiKey:${API_KEY_ID}`));

    await prisma.cohort.upsert({
        where: { orgId_id: { orgId: ORG.id, id: COHORT_ID } },
        update: { name: 'Signup Date', type: COHORT_ID },
        create: { orgId: ORG.id, id: COHORT_ID, name: 'Signup Date', type: COHORT_ID },
    }).catch(ignoreConflict(`cohort:${COHORT_ID}`));

    console.log('Seed complete.');
}

main()
    .catch((e) => {
        console.error(e);
        process.exit(1);
    })
    .finally(async () => {
        await prisma.$disconnect();
    });
