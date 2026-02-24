"use client";

import { motion } from "motion/react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import LightRays from "./LightRays";

export default function Home() {
  const router = useRouter();

  return (
    <div className="relative min-h-screen w-full overflow-hidden bg-gradient-to-b from-black/75 via-neutral-900/75 to-black/75 text-white">
      <div className="pointer-events-none fixed inset-0 z-0">
        <div style={{ width: "100%", height: "100vh", position: "relative" }}>
          <LightRays
            raysOrigin="top-center"
            raysColor="#ffffff"
            raysSpeed={1}
            lightSpread={0.5}
            rayLength={3}
            followMouse={true}
            mouseInfluence={0.1}
            noiseAmount={0}
            distortion={0}
            className="custom-rays"
            pulsating={false}
            fadeDistance={1}
            saturation={1}
          />
        </div>
      </div>

      <div className="relative z-10">
        {/* Top Bar */}
        <div className="flex justify-between items-center p-6 sticky top-0 z-50 backdrop-blur-md bg-black/50">
          <Button
            size="sm"
            variant="secondary"
            onClick={() => router.push("/dashboard")}
          >
            Dashboard
          </Button>

          <div className="flex gap-3">
            <Button size="sm" onClick={() => router.push("/login")}>
              Login
            </Button>
            <Button
              size="sm"
              variant="secondary"
              onClick={() => router.push("/setting")}
            >
              Settings
            </Button>
          </div>
        </div>

        {/* Hero Section */}
        <section className="flex flex-col items-center justify-center text-center px-6 py-32">
          <motion.h1
            initial={{ opacity: 0, y: 40 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.8 }}
            className="text-4xl md:text-7xl font-bold mb-6"
          >
            Stay Connected. Stay Safe.
          </motion.h1>

          <p className="text-lg md:text-2xl font-light max-w-2xl mb-10 text-neutral-300">
            Real-time location sharing that helps you stay connected with the
            people who matter most.
          </p>

          <Button
            className="w-60 text-lg"
            onClick={() => alert("Location feature coming soon")}
          >
            Try Live Location
          </Button>
        </section>

      {/* Benefits Section */}
      <section className="px-6 py-24 max-w-6xl mx-auto grid md:grid-cols-3 gap-10 text-center">
        <div className="bg-white/5 backdrop-blur-lg rounded-2xl p-8 border border-white/10">
          <h3 className="text-2xl font-semibold mb-4">
            Keep an Eye on Your Kids
          </h3>
          <p className="text-neutral-400">
            Know when they reach school, arrive home safely, or head out with
            friends. Get peace of mind without constant phone calls.
          </p>
        </div>

        <div className="bg-white/5 backdrop-blur-lg rounded-2xl p-8 border border-white/10">
          <h3 className="text-2xl font-semibold mb-4">
            Ensure Elder Safety
          </h3>
          <p className="text-neutral-400">
            Stay reassured knowing your elderly family members are safe and
            where they should be. Quick location access during emergencies.
          </p>
        </div>

        <div className="bg-white/5 backdrop-blur-lg rounded-2xl p-8 border border-white/10">
          <h3 className="text-2xl font-semibold mb-4">
            Real-Time Updates
          </h3>
          <p className="text-neutral-400">
            Instantly see live location updates, movement status, and shared
            circles — all in one place.
          </p>
        </div>
      </section>

      {/* Extra Section */}
      <section className="px-6 py-24 text-center max-w-4xl mx-auto">
        <h2 className="text-3xl md:text-5xl font-bold mb-8">
          Built for Families
        </h2>
        <p className="text-lg text-neutral-400 leading-relaxed">
          Whether you&apos;re coordinating pickups, checking that loved ones are
          safe, or simply staying connected throughout the day, this platform
          gives you a secure and private way to share location information.
          Your safety and privacy remain the top priority.
        </p>
      </section>
	
            {/* Smart Features Section */}
      <section className="px-6 py-24 max-w-6xl mx-auto grid md:grid-cols-2 gap-16 items-center">
        <div>
          <h2 className="text-3xl md:text-5xl font-bold mb-6">
            Smart Alerts & Notifications
          </h2>
          <p className="text-neutral-400 text-lg leading-relaxed">
            Set custom alerts for important places like home, school, or work.
            Receive notifications when loved ones arrive or leave specific
            locations. Stay informed without constantly checking the app.
          </p>
        </div>

        <div className="bg-white/5 border border-white/10 rounded-2xl p-8 backdrop-blur-lg">
          <ul className="space-y-4 text-neutral-300">
            <li>• Arrival & departure alerts</li>
            <li>• Low battery notifications</li>
            <li>• Real-time movement updates</li>
            <li>• Custom safe zones</li>
          </ul>
        </div>
      </section>

      {/* Privacy Section */}
      <section className="px-6 py-24 text-center max-w-4xl mx-auto">
        <h2 className="text-3xl md:text-5xl font-bold mb-8">
          Privacy First. Always.
        </h2>
        <p className="text-lg text-neutral-400 leading-relaxed">
          Your location data is shared only with people you trust. You stay in
          control of who sees your location and when. Designed with security in
          mind, so your family stays protected.
        </p>
      </section>

      {/* Emergency Section */}
      <section className="px-6 py-24 max-w-5xl mx-auto text-center">
        <div className="bg-red-500/10 border border-red-500/30 rounded-2xl p-12 backdrop-blur-lg">
          <h2 className="text-3xl md:text-4xl font-bold mb-6">
            Emergency Assistance
          </h2>
          <p className="text-neutral-300 text-lg">
            Quickly share your real-time location in urgent situations.
            Immediate access to your location ensures faster help when it
            matters most.
          </p>
        </div>
      </section>

      {/* Cross Device Section */}
      <section className="px-6 py-24 text-center max-w-4xl mx-auto">
        <h2 className="text-3xl md:text-5xl font-bold mb-8">
          Works Across Devices
        </h2>
        <p className="text-lg text-neutral-400 leading-relaxed">
          Access your dashboard from desktop, tablet, or mobile. Stay connected
          wherever you are with seamless synchronization.
        </p>
      </section>

        {/* FAQ Section */}
        <section className="px-6 py-24 max-w-4xl mx-auto">
          <h2 className="text-3xl md:text-5xl font-bold mb-12 text-center">
            Frequently Asked Questions
          </h2>

          <div className="space-y-8">
            <div>
              <h3 className="text-xl font-semibold mb-2">
                Does this track people without permission?
              </h3>
              <p className="text-neutral-400">
                No. Location sharing requires mutual consent. Users control who
                can see their location.
              </p>
            </div>

            <div>
              <h3 className="text-xl font-semibold mb-2">
                Is location sharing real-time?
              </h3>
              <p className="text-neutral-400">
                Yes. Locations update dynamically to give you accurate live
                tracking.
              </p>
            </div>

            <div>
              <h3 className="text-xl font-semibold mb-2">
                Can I turn off location anytime?
              </h3>
              <p className="text-neutral-400">
                Absolutely. You are always in control of your privacy settings.
              </p>
            </div>
          </div>
        </section>
      </div>
    </div>
  );
}
