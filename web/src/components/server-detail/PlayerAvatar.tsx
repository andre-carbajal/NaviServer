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
    className="server-v2-player-avatar"
    onError={(event) => {
      const fallbackUrl = getAvatarUrl();
      if (event.currentTarget.src !== fallbackUrl) {
        event.currentTarget.src = fallbackUrl;
      }
    }}
  />
);

export default PlayerAvatar;
