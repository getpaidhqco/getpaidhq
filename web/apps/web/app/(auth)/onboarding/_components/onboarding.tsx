"use client"

import {useRouter} from 'next/navigation'
import {useAuth} from '@getpaidhq/auth'
import {Input} from '@/components/ui/input'
import {Button} from '@/components/ui/button'
import {Heading} from '@/components/atoms/heading'
import Link from 'next/link'
import {P as Text} from '@/components/ui/typography'
import {useForm} from "react-hook-form";
import {zodResolver} from "@hookform/resolvers/zod";
import {z} from "zod";
import {toast} from "sonner";
import type React from "react";
import {cn} from "@/lib/utils";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";


const Schema = z.object({
  name: z.string().min(3),
  country: z.string().optional().nullable(),
  timezone: z.string().optional().nullable(),
})
type SchemaType = z.infer<typeof Schema>;

export default function OnboardingPage() {
  const router = useRouter()
  const {getAuthHeaders, reloadSession, setActiveOrg} = useAuth()

  const form = useForm<SchemaType>({
    resolver: zodResolver(Schema),
    defaultValues: {
      name: "",
      country: "ZA",
      timezone: "Africa/Johannesburg",
    },
  });

  const onSubmit = async (values: SchemaType) => {
    const rsp = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/organizations`, {
      headers: await getAuthHeaders(),
      method: 'POST',
      body: JSON.stringify(values),
    })
    if (!rsp.ok) {
      const error = await rsp.json();
      console.error(error);
      toast.error(`An error occurred with the onboarding, please contact support to complete your signup.`)
      return;
    }
    const data = await rsp.json();

    console.log('Organization created:', data);
    await setActiveOrg(data.id)
    await reloadSession();
    router.push('/dashboard')
  };

  const isSubmitting = form.formState.isSubmitting;

  return (
    <div className="flex min-h-dvh flex-col p-2">
      <div className="flex items-center justify-center min-h-screen bg-background p-4">
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="grid w-full max-w-sm grid-cols-1 gap-8">
            <div>
              <Heading>Welcome to GetPaidHQ</Heading>
              <Text>Please complete the form below to set up your account.</Text>
            </div>
            <FormField
              control={form.control}
              name="name"
              render={({field}) => (
                <FormItem>
                  <FormLabel>Organization Name</FormLabel>
                  <FormControl>
                    <Input {...field} />
                  </FormControl>
                  <FormMessage/>
                </FormItem>
              )}
            />

            <Button
              type="submit"
              className={cn(
                isSubmitting && "opacity-50 cursor-not-allowed",
                "w-full bg-primary"
              )}
              disabled={isSubmitting}
            >
              {isSubmitting ? 'Creating...' : 'Create Organization'}
            </Button>
            <Text>
              Don’t have an account?{' '}
              <Link href="#" className="text-primary underline-offset-4 hover:underline">
                <strong className="font-semibold">Sign up</strong>
              </Link>
            </Text>
          </form>
        </Form>
      </div>
    </div>

  )
}
