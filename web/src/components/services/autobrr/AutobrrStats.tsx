/*
 * Copyright (c) 2024, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { useServiceData } from "../../../hooks/useServiceData";
import { StatusIcon } from "../../ui/StatusIcon";
import { AutobrrMessage } from "./AutobrrMessage";

interface AutobrrStatsProps {
  instanceId: string;
}

export const AutobrrStats: React.FC<AutobrrStatsProps> = ({ instanceId }) => {
  const { services } = useServiceData();
  const service = services.find((s) => s.instanceId === instanceId);
  const isLoading = service?.status === "loading";

  if (isLoading) {
    return (
      <div className="space-y-3">
        {[1, 2, 3].map((i) => (
          <div
            key={i}
            className="flex items-center space-x-3 bg-gray-50 dark:bg-gray-700/50 p-3 rounded-lg animate-pulse"
          >
            <div className="min-w-0 flex-1">
              <div className="h-4 bg-gray-200 dark:bg-gray-600 rounded w-3/4 mb-2" />
              <div className="flex space-x-2">
                <div className="h-3 bg-gray-200 dark:bg-gray-600 rounded w-20" />
                <div className="h-3 bg-gray-200 dark:bg-gray-600 rounded w-24" />
              </div>
            </div>
            <div className="flex-shrink-0">
              <div className="h-4 bg-gray-200 dark:bg-gray-600 rounded w-16" />
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (!service) {
    return null;
  }

  // Always show stats section if service is online, even if stats are empty
  const showStats = true;
  const stats = service.stats?.autobrr || {
    total_count: 0,
    filtered_count: 0,
    filter_rejected_count: 0,
    push_approved_count: 0,
    push_rejected_count: 0,
    push_error_count: 0,
  };
  const ircStatus = service.details?.autobrr?.irc;

  // Only show message component if there's a message or status isn't online
  const showMessage = service.message || service.status !== "online";

  return (
    <div className="space-y-4">
      {/* Status and Messages */}
      {showMessage && (
        <AutobrrMessage status={service.status} message={service.message} />
      )}

      {/* IRC Status */}
      {ircStatus && (
        <div>
          <div className="text-xs mb-2 font-semibold text-gray-700 dark:text-gray-300">
            IRC Status:
          </div>
          <div className="text-sm rounded-md text-gray-600 dark:text-gray-400 bg-gray-850/95 p-4 space-y-1">
            {ircStatus.map((irc, index) => (
              <div key={index} className="flex justify-between items-center">
                <span className="font-medium">{irc.name}</span>
                <div className="flex items-center">
                  <StatusIcon status={irc.healthy ? "online" : "error"} />
                  <span className="ml-2" style={{ color: "inherit" }}>
                    {irc.healthy ? "Healthy" : "Unhealthy"}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Stats */}
      {showStats && (
        <div>
          <div className="text-xs mb-2 font-semibold text-gray-700 dark:text-gray-300">
            Stats:
          </div>
          <div className="grid grid-cols-2 gap-2">
            <div className="text-xs rounded-md text-gray-600 dark:text-gray-400 bg-gray-850/95 p-4">
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm font-medium">Filtered Releases:</span>
              </div>
              <div className="font-bold">{stats.filtered_count || 0}</div>
            </div>

            <div className="text-xs rounded-md text-gray-600 dark:text-gray-400 bg-gray-850/95 p-4">
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm font-medium">Approved Pushes:</span>
              </div>
              <div className="font-bold">{stats.push_approved_count || 0}</div>
            </div>

            <div className="text-xs rounded-md text-gray-600 dark:text-gray-400 bg-gray-850/95 p-4">
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm font-medium">Rejected Pushes:</span>
              </div>
              <div className="font-bold">{stats.push_rejected_count || 0}</div>
            </div>

            <div className="text-xs rounded-md text-gray-600 dark:text-gray-400 bg-gray-850/95 p-4">
              <div className="flex items-center justify-between mb-2">
                <span className="text-sm font-medium">Errored Pushes:</span>
              </div>
              <div className="font-bold">{stats.push_error_count || 0}</div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
