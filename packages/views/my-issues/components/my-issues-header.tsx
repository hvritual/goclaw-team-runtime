"use client";

import { ChevronDown } from "lucide-react";
import { Button } from "@multica/ui/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@multica/ui/components/ui/dropdown-menu";
import { Tooltip, TooltipTrigger, TooltipContent } from "@multica/ui/components/ui/tooltip";
import type {
  Issue,
  IssueTableFacetSpec,
  IssueTableFacetsResponse,
} from "@multica/core/types";
import type { MyIssuesScope } from "@multica/core/issues/stores/my-issues-view-store";
import { useT } from "../../i18n";
import {
  IssueDisplayControls,
  ViewRefreshIndicator,
} from "../../issues/components/issues-header";

export function MyIssuesHeader({
  allIssues,
  scope,
  onScopeChange,
  isRefreshing = false,
  facetCountsExact = true,
  tableFacetCounts,
  onTableFacetChange,
}: {
  allIssues: Issue[];
  scope: MyIssuesScope;
  onScopeChange: (scope: MyIssuesScope) => void;
  isRefreshing?: boolean;
  /** See IssueDisplayControls.facetCountsExact. */
  facetCountsExact?: boolean;
  tableFacetCounts?: IssueTableFacetsResponse;
  onTableFacetChange: (facet: IssueTableFacetSpec | null) => void;
}) {
  const { t } = useT("my-issues");
  const SCOPES: { value: MyIssuesScope; label: string; description: string }[] = [
    { value: "all", label: t(($) => $.header.scope.all_label), description: t(($) => $.header.scope.all_description) },
    { value: "assigned", label: t(($) => $.header.scope.assigned_label), description: t(($) => $.header.scope.assigned_description) },
    { value: "created", label: t(($) => $.header.scope.created_label), description: t(($) => $.header.scope.created_description) },
  ];
  const scopeLabel = SCOPES.find((s) => s.value === scope)?.label ?? SCOPES[0]?.label;

  return (
    <div className="h-12 shrink-0 overflow-x-auto px-4 [-webkit-overflow-scrolling:touch]">
      <div className="flex h-full w-max min-w-full items-center justify-between gap-2">
        <div className="hidden shrink-0 items-center gap-1 md:flex">
          {SCOPES.map((s) => (
            <Tooltip key={s.value}>
              <TooltipTrigger
                render={
                  <Button
                    variant="outline"
                    size="sm"
                    className={
                      scope === s.value
                        ? "bg-accent text-accent-foreground hover:bg-accent/80"
                        : "text-muted-foreground"
                    }
                    onClick={() => onScopeChange(s.value)}
                  >
                    {s.label}
                  </Button>
                }
              />
              <TooltipContent side="bottom">{s.description}</TooltipContent>
            </Tooltip>
          ))}
        </div>

        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                variant="outline"
                size="sm"
                className="shrink-0 gap-1 text-muted-foreground md:hidden"
              >
                <span className="truncate">{scopeLabel}</span>
                <ChevronDown className="size-3 text-muted-foreground" />
              </Button>
            }
          />
          <DropdownMenuContent align="start" className="w-auto">
            <DropdownMenuRadioGroup
              value={scope}
              onValueChange={(value) => onScopeChange(value as MyIssuesScope)}
            >
              {SCOPES.map((s) => (
                <DropdownMenuRadioItem key={s.value} value={s.value}>
                  {s.label}
                </DropdownMenuRadioItem>
              ))}
            </DropdownMenuRadioGroup>
          </DropdownMenuContent>
        </DropdownMenu>

        <div className="flex shrink-0 items-center gap-1">
          <IssueDisplayControls
            scopedIssues={allIssues}
            facetCountsExact={facetCountsExact}
            tableFacetCounts={tableFacetCounts}
            onTableFacetChange={onTableFacetChange}
          />
          <ViewRefreshIndicator active={isRefreshing} />
        </div>
      </div>
    </div>
  );
}
