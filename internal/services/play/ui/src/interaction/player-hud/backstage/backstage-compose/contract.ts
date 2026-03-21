export type BackstageComposeProps = {
  draft: string;
  disabled?: boolean;
  onDraftChange: (value: string) => void;
  onSend: () => void;
};
