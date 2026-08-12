"use client";

import { use } from "react";
import { TeamControlPage } from "@multica/views/team-control";

export default function ProjectTeamControlPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  return <TeamControlPage projectId={id} />;
}
