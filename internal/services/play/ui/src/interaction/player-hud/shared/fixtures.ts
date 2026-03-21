import { participantAvatarPreviewAssets } from "../../../storybook/preview-assets/fixtures";
import { backstageFixtureCatalog, backstageParticipants } from "../backstage/shared/fixtures";
import { playerHUDCharacterInspectionCatalog } from "./character-inspection-fixtures";
import {
  onStageCharacterCatalog,
  onStageFixtureCatalog,
  onStageParticipants,
} from "../on-stage/shared/fixtures";
import type { PlayerHUDState, SideChatMessage, SideChatParticipant, SideChatState } from "./contract";

export const sideChatParticipants: SideChatParticipant[] = [
  backstageParticipants[0],
  { ...backstageParticipants[1], typing: true },
  backstageParticipants[2],
];

export const sideChatMessages: SideChatMessage[] = [
  { id: "m1", participantId: "p-bryn", body: "Ready when you are.", sentAt: "2026-03-18T16:30:00Z" },
  { id: "m2", participantId: "p-bryn", body: "I'll take the left flank.", sentAt: "2026-03-18T16:30:15Z" },
  { id: "m3", participantId: "p-rhea", body: "Copy. Moving to the bridge.", sentAt: "2026-03-18T16:31:00Z" },
  { id: "m4", participantId: "p-guide", body: "Quick heads-up: I'm adding a weather complication next round.", sentAt: "2026-03-18T16:32:00Z" },
  { id: "m5", participantId: "p-rhea", body: "Sounds good!", sentAt: "2026-03-18T16:32:30Z" },
  { id: "m6", participantId: "p-rhea", body: "Should we prep anything?", sentAt: "2026-03-18T16:32:45Z" },
];

export const sideChatState: SideChatState = {
  viewerParticipantId: "p-rhea",
  participants: sideChatParticipants,
  messages: sideChatMessages,
  characterInspectionCatalog: playerHUDCharacterInspectionCatalog,
};

export const emptySideChatState: SideChatState = {
  viewerParticipantId: "p-rhea",
  participants: sideChatParticipants,
  messages: [],
  characterInspectionCatalog: playerHUDCharacterInspectionCatalog,
};

const ivesAvatar = participantAvatarPreviewAssets[3];

const onStageShellPreview = {
  ...onStageFixtureCatalog.viewerPosted,
  participants: [
    ...onStageFixtureCatalog.viewerPosted.participants.map((participant) =>
      participant.id === "p-rhea" || participant.id === "p-bryn"
        ? { ...participant, railStatus: "active" as const }
        : participant
    ),
    {
      id: "p-ives",
      name: "Ives",
      role: "player" as const,
      avatarUrl: ivesAvatar?.imageUrl,
      characters: [onStageCharacterCatalog.sable],
      railStatus: "active" as const,
    },
  ],
  actingParticipantIds: ["p-rhea", "p-bryn", "p-ives"],
  actingCharacterNames: ["Aria", "Corin", "Sable"],
  sceneDescription:
    "The drowned chapel vault crouches beneath a leaning bell tower, its black-stone door slick with condensation while hairline sigils pulse under the bronze seam like banked lightning. Water from the nave has crept down the steps in a glass-thin sheet, and every carved saint in the corridor has been worn faceless except for the one directly above the lock, whose remaining mouth seems to split wider whenever the ward hum rises. A rack of rusted censers swings somewhere out of sight, tapping faintly against stone in the same off-tempo rhythm as the lower glyphs.",
  gmOutputText:
    "Lanternlight catches on the ward lattice every time Aria leans in. Corin spots three glyphs near the lower hinge flashing out of rhythm, but each pulse is answered by a distant iron knock somewhere deeper in the keep. Sable, posted halfway up the flooded stairs with one eye on the transept, hears armored movement stop and start above the chapel floorboards as if someone is listening for the ward to break cleanly. The cold spilling through the seam smells of wet parchment, extinguished incense, and a tomb that has been opened once already tonight.",
  frameText:
    "The seal has parted by barely a finger's width, enough to breathe out cold dust and stale incense. Aria, if you commit to the pry, what exact leverage do you take, and Corin, how do you keep the ward from translating that motion into a full alarm through the chapel above? Sable, you are the only one with a clear line to the upper gallery, so tell me how you manage the patrol's approach, the hanging censers, and the one stretch of open water that will throw back lanternlight if anyone moves too quickly. If this goes wrong, I need to know which compromise each of you is already prepared to make: noise, delay, or exposure.",
  slots: [
    ...onStageFixtureCatalog.viewerPosted.slots.map((slot) => {
      if (slot.participantId === "p-rhea") {
        return {
          ...slot,
          body:
            "Aria braces one boot against the flooded threshold, slides the pry tool into the one place the bronze lip has already warped, and leans only until the seam groans. She keeps her left hand off the door entirely, shoulder turned so the recoil will glance past her instead of through her chest, and counts under her breath with Corin's rhythm before she commits the next inch of pressure. When the metal answers with a low complaint, she pauses, shifts the tool half a finger lower, and forces herself to work with patience instead of force.",
        };
      }

      if (slot.participantId === "p-bryn") {
        return {
          ...slot,
          body:
            "Corin crouches beside the hinge with the lantern hooded to a sliver, tracking the pulse pattern in a whispered count. He marks each weak point against the stone with the butt of a chalk nub, then wipes two of them away when the lattice shifts and the safe pattern starts drifting upslope. When the lower sigil stutters, he taps the surviving beat against the threshold and guides Aria toward the one moment where the ward's recoil is weakest, already preparing a second warning if the knock from deeper in the keep resolves into a lock turning upstairs.",
        };
      }

      return slot;
    }),
    {
      id: "p-ives-char-sable",
      participantId: "p-ives",
      characters: [onStageCharacterCatalog.sable],
      body:
        "Sable keeps low along the chapel stairs with one dagger laid flat against the stone to stop its fittings from chiming. From the shadow of the broken font, he watches the gallery rail for movement, times the sway of the hanging censers, and nudges a loose scrap of hymn parchment into the open water so he can see whether any new draft or footfall is coming before the patrol rounds the arch.",
      updatedAt: "2026-03-19T20:32:40Z",
      yielded: false,
      reviewState: "open" as const,
    },
  ],
  characterInspectionCatalog: playerHUDCharacterInspectionCatalog,
};

export const playerHUDFixtureCatalog: Record<
  "onStage" | "backstage" | "sideChat",
  PlayerHUDState
> = {
  onStage: {
    activeTab: "on-stage",
    onStage: onStageShellPreview,
    backstage: backstageFixtureCatalog.dormant,
    sideChat: sideChatState,
  },
  backstage: {
    activeTab: "backstage",
    onStage: onStageFixtureCatalog.waitingOnGM,
    backstage: backstageFixtureCatalog.openDiscussion,
    sideChat: sideChatState,
  },
  sideChat: {
    activeTab: "side-chat",
    onStage: onStageFixtureCatalog.aiThinking,
    backstage: backstageFixtureCatalog.waitingOnGM,
    sideChat: sideChatState,
  },
};

export {
  backstageFixtureCatalog,
  backstageParticipants,
  onStageCharacterCatalog,
  onStageFixtureCatalog,
  onStageParticipants,
};
