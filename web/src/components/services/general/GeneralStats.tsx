/*
 * Copyright (c) 2024, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import React from "react";
import { useServiceData } from "../../../hooks/useServiceData";
import { GeneralMessage } from "./GeneralMessage";

interface GeneralStatsProps {
  instanceId: string;
}

export const GeneralStats: React.FC<GeneralStatsProps> = ({ instanceId }) => {
  const { services } = useServiceData();
  const service = services.find((s) => s.instanceId === instanceId);
  const isLoading = service?.status === "loading";

  if (isLoading) {
    return (
      <div className="space-y-3">
        <div className="flex items-center space-x-3 bg-gray-50 dark:bg-gray-700/50 p-3 rounded-lg animate-pulse">
          <div className="min-w-0 flex-1">
            <div className="h-4 bg-gray-200 dark:bg-gray-600 rounded w-3/4 mb-2" />
            <div className="flex space-x-2">
              <div className="h-3 bg-gray-200 dark:bg-gray-600 rounded w-20" />
              <div className="h-3 bg-gray-200 dark:bg-gray-600 rounded w-24" />
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (!service) {
    return null;
  }

  // Only show message component if there's a message or status isn't online
  const showMessage = service.message || service.status !== "online";

  return (
    <div className="space-y-4">
      {showMessage && (
        <GeneralMessage status={service.status} message={service.message} />
      )}
    </div>
  );
};
