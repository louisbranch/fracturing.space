import type { JSX } from "react";
import type { PlayBootstrap, PlayRoomSnapshot } from "./protocol";
import type { SystemRenderViewModel } from "./view_models";

export type SystemRendererProps = {
  bootstrap: PlayBootstrap;
  snapshot: PlayRoomSnapshot;
  view: SystemRenderViewModel;
};

export interface SystemRenderer {
  id: string;
  render(props: SystemRendererProps): JSX.Element;
}
