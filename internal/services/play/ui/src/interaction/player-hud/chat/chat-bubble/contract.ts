export type ChatBubbleProps = {
  body: string;
  time: string; // pre-formatted "hh:mm"
  alignment: "start" | "end";
  showName?: string; // participant name, only on first in run
  showAvatar?: boolean; // only on last in run
  avatarUrl?: string;
  avatarFallback?: string; // first letter of name
};
