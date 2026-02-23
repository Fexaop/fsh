"use client";

import { useRouter } from "next/navigation";
import { useState, useEffect } from "react";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";

interface Member {
  name: string;
  image: string | null;
}

export default function SettingsPage() {
  const router = useRouter();

  const [username, setUsername] = useState("");
  const [profilePic, setProfilePic] = useState<string | null>(null);

  const [members, setMembers] = useState<Member[]>([]);
  const [newMemberName, setNewMemberName] = useState("");
  const [newMemberImage, setNewMemberImage] = useState<string | null>(null);

  useEffect(() => {
    const storedName = localStorage.getItem("username");
    const storedPic = localStorage.getItem("profilePic");
    const storedMembers = localStorage.getItem("members");

    if (storedName) setUsername(storedName);
    if (storedPic) setProfilePic(storedPic);
    if (storedMembers) setMembers(JSON.parse(storedMembers));
  }, []);

  const handleProfileImageUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onloadend = () => {
      setProfilePic(reader.result as string);
    };
    reader.readAsDataURL(file);
  };

  const handleMemberImageUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onloadend = () => {
      setNewMemberImage(reader.result as string);
    };
    reader.readAsDataURL(file);
  };

  const handleAddMember = () => {
    if (!newMemberName) return;

    const updatedMembers = [
      ...members,
      { name: newMemberName, image: newMemberImage },
    ];

    setMembers(updatedMembers);
    localStorage.setItem("members", JSON.stringify(updatedMembers));

    setNewMemberName("");
    setNewMemberImage(null);
  };

  const handleSaveProfile = () => {
    localStorage.setItem("username", username);
    if (profilePic) {
      localStorage.setItem("profilePic", profilePic);
    }
  };

  return (
    <div className="flex justify-center items-center min-h-screen bg-muted/40 p-4">
      <Card className="w-full max-w-md shadow-lg">
        <CardHeader className="text-center">
          <CardTitle>⚙ Settings</CardTitle>
          <CardDescription>
            Manage your profile and family members
          </CardDescription>
        </CardHeader>

        <CardContent className="space-y-8">

          {/* PROFILE SECTION */}
          <div className="space-y-4">
            <Label>Username</Label>
            <Input
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="Enter username"
            />

            <Label>Profile Picture</Label>
            <Input
              type="file"
              accept="image/*"
              onChange={handleProfileImageUpload}
            />

            {profilePic && (
              <div className="flex justify-center">
                <img
                  src={profilePic}
                  alt="Profile"
                  className="w-24 h-24 rounded-full object-cover border"
                />
              </div>
            )}

            <Button onClick={handleSaveProfile} className="w-full">
              Save Profile
            </Button>
          </div>

          {/* MEMBERS SECTION */}
          <div className="space-y-4 border-t pt-6">
            <h3 className="text-lg font-semibold text-center">
              👨‍👩‍👧 Family Members
            </h3>

            <Input
              placeholder="Member Name"
              value={newMemberName}
              onChange={(e) => setNewMemberName(e.target.value)}
            />

            <Input
              type="file"
              accept="image/*"
              onChange={handleMemberImageUpload}
            />

            {newMemberImage && (
              <div className="flex justify-center">
                <img
                  src={newMemberImage}
                  alt="Preview"
                  className="w-20 h-20 rounded-full object-cover border"
                />
              </div>
            )}

            <Button onClick={handleAddMember} className="w-full">
              Add Member
            </Button>

            {/* MEMBERS LIST */}
            <div className="space-y-3 mt-4">
              {members.map((member, index) => (
                <div
                  key={index}
                  className="flex items-center gap-4 p-3 border rounded-lg"
                >
                  {member.image && (
                    <img
                      src={member.image}
                      alt={member.name}
                      className="w-10 h-10 rounded-full object-cover"
                    />
                  )}
                  <span className="font-medium">{member.name}</span>
                </div>
              ))}
            </div>
          </div>

          <Button
            variant="secondary"
            className="w-full"
            onClick={() => router.push("/dashboard")}
          >
            Back to Dashboard
          </Button>

        </CardContent>
      </Card>
    </div>
  );
}