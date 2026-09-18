import React from 'react';

import type { PlayerInfo } from '../../types';
import { getAvatarUrl } from '../../utils/serverDetail';

interface PlayerAvatarProps {
  player: PlayerInfo;
}

const PlayerAvatar: React.FC<PlayerAvatarProps> = ({ player }) => (
  <img
    src={getAvatarUrl(player.id)}
    alt={`${player.name} avatar`}
    className="tw:h-7 tw:w-7 tw:rounded-md tw:border tw:border-border tw:bg-white/5 [image-rendering:pixelated]"
    onError={(event) => {
      const fallbackUrl = getAvatarUrl();
      if (event.currentTarget.src !== fallbackUrl) {
        event.currentTarget.src = fallbackUrl;
      }
    }}
  />
);

export default PlayerAvatar;
