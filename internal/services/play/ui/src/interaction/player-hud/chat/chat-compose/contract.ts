export type ChatComposeProps = {
  draft: string;
  onDraftChange: (value: string) => void;
  onSend: () => void;
  disabled?: boolean;
  ariaLabel?: string;
  placeholder?: string;
  sendLabel?: string;
};
