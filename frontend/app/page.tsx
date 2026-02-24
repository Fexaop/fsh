import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { SESSION_TOKEN_COOKIE } from "@/lib/auth";

export default async function Home() {
  const cookieStore = await cookies();
  const isAuthenticated = Boolean(cookieStore.get(SESSION_TOKEN_COOKIE)?.value);

  if (isAuthenticated) {
    redirect("/dashboard");
  }

  redirect("/home");
}
