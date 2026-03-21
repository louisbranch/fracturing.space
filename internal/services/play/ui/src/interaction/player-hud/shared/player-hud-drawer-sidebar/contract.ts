import type { PlayerHUDCampaignNavigation } from "../contract";

export type PlayerHUDDrawerSidebarProps = {
  navigation: PlayerHUDCampaignNavigation;
  onCharacterInspect?: (participantId: string, characterId: string) => void;
  onClose: () => void;
};
