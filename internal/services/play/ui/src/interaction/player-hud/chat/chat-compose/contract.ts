export type ChatComposeProps = {
  draft: string;
  onDraftChange: (value: string) => void;
  onSend: () => void;
};
