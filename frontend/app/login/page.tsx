"use client";
import { motion } from "motion/react";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

import { AuroraBackground } from "@/components/ui/aurora-background";
import { Button } from "@/components/ui/button";
import {
  BACKEND_BASE_URL,
  type GoogleAuthSessionResponse,
  type GoogleAuthStartResponse,
  hasSessionToken,
  setSessionTokenCookie,
} from "@/lib/auth";

const AUTH_POLL_INTERVAL_MS = 1500;
const AUTH_POLL_TIMEOUT_MS = 10 * 60 * 1000;

export default function Login() {
  const router = useRouter();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  useEffect(() => {
    if (hasSessionToken(document.cookie)) {
      router.replace("/dashboard");
    }
  }, [router]);

  const handleLogin = async () => {
    if (isSubmitting) {
      return;
    }

    setIsSubmitting(true);
    setErrorMessage(null);

    try {
      const startResponse = await fetch(`${BACKEND_BASE_URL}/auth/google/start`);
      if (!startResponse.ok) {
        throw new Error("Unable to start Google auth.");
      }

      const { authUrl, state } =
        (await startResponse.json()) as GoogleAuthStartResponse;
      if (!authUrl || !state) {
        throw new Error("Backend returned an invalid Google auth response.");
      }

      const popup = window.open(
        authUrl,
        "fsh-google-auth",
        "popup=yes,width=520,height=680",
      );

      if (!popup) {
        window.location.href = authUrl;
        return;
      }

      const deadline = Date.now() + AUTH_POLL_TIMEOUT_MS;

      while (Date.now() < deadline) {
        const sessionResponse = await fetch(
          `${BACKEND_BASE_URL}/auth/google/session?state=${encodeURIComponent(state)}`,
        );
        if (!sessionResponse.ok) {
          throw new Error("Failed to verify Google login.");
        }

        const sessionData =
          (await sessionResponse.json()) as GoogleAuthSessionResponse;

        if (
          sessionData.status === "authenticated" &&
          sessionData.sessionToken
        ) {
          setSessionTokenCookie(sessionData.sessionToken);
          popup.close();
          router.replace("/dashboard");
          return;
        }

        if (
          sessionData.status === "expired" ||
          sessionData.status === "not_found"
        ) {
          throw new Error("Google login expired. Please try again.");
        }

        if (sessionData.status === "pending" && popup.closed) {
          throw new Error("Google login window was closed before completion.");
        }

        await new Promise((resolve) =>
          window.setTimeout(resolve, AUTH_POLL_INTERVAL_MS),
        );
      }

      throw new Error("Google login timed out. Please try again.");
    } catch (error) {
      const message =
        error instanceof Error ? error.message : "Google login failed.";
      setErrorMessage(message);
      setIsSubmitting(false);
    }
  };

  return (
    <AuroraBackground>
      <motion.div
        initial={{ opacity: 0.0, y: 40 }}
        whileInView={{ opacity: 1, y: 0 }}
        transition={{
          delay: 0.3,
          duration: 0.8,

          ease: "easeInOut",
        }}
        className="relative flex flex-col gap-4 items-center justify-center px-4"
      >
        <div className="text-3xl md:text-7xl font-bold dark:text-white text-center">
          Find my - connect with friends.
        </div>
        <div className="font-extralight text-base md:text-4xl dark:text-neutral-200 py-4">
          login/signup to get started.
        </div>
        <Button onClick={handleLogin} disabled={isSubmitting}>
          {isSubmitting ? "Signing in with Google..." : "Login with Google"}
        </Button>
        {errorMessage ? (
          <p className="text-sm text-red-600 dark:text-red-400">{errorMessage}</p>
        ) : null}
      </motion.div>
    </AuroraBackground>
  );
}
