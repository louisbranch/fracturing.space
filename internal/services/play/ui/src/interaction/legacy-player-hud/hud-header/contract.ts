import type { PlayerConnectionState } from "../shared/contract";

export type HUDHeaderProps = {
  campaignName: string;
  backURL: string;
  connection: PlayerConnectionState;
};
