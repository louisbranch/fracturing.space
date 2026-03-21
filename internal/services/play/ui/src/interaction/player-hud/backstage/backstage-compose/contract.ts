export type BackstageComposeProps = {
  draft: string;
  viewerReady: boolean;
  disabled?: boolean;
  onDraftChange: (value: string) => void;
  onSend: () => void;
  onReadyToggle: () => void;
};
