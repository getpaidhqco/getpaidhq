import {NextRequest, NextResponse} from 'next/server'
import {loadAuthProvider} from "@getpaidhq/auth/server";


const auth = loadAuthProvider();

export async function POST(req: NextRequest) {
  try {
    // Get the authenticated user
    const user = await auth.auth(req);

    if (!user) {
      return NextResponse.json({error: 'Unauthorized'}, {status: 401})
    }

    // Parse the request body
    const body = await req.json()

    // Validate the request body
    if (!body.name) {
      return NextResponse.json({error: 'Organization name is required'}, {status: 400})
    }

    // Create a new organization
    const orgId = `org_${Date.now()}`
    const organization = {
      id: orgId,
      name: body.name,
      description: body.description || '',
      createdBy: user.id,
      members: [user.id],
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString()
    }


    // Associate the user with the organization
    // In a real application, you would update the user record in your database

    return NextResponse.json(organization, {status: 201})
  } catch (error) {
    console.error('Error creating organization:', error)
    return NextResponse.json({error: 'Internal server error'}, {status: 500})
  }
}
