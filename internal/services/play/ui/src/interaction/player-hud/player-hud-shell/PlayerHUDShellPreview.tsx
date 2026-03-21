import { useState } from "react";
import {
  PlayerHUDCharacterInspectorDialog,
  usePlayerHUDCharacterInspector,
} from "../shared/PlayerHUDCharacterInspector";
import type { BackstageParticipant } from "../backstage/shared/contract";
import type { OnStageState } from "../on-stage/shared/contract";
import type { HUDNavbarTab, PlayerHUDState } from "../shared/contract";
import { PlayerHUDShell } from "./PlayerHUDShell";

type PlayerHUDShellPreviewProps = {
  initialState: PlayerHUDState;
};

export function PlayerHUDShellPreview({ initialState }: PlayerHUDShellPreviewProps) {
  const [activeTab, setActiveTab] = useState<HUDNavbarTab>(initialState.activeTab);
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);
  const initialOnStageDraft =
    initialState.onStage.slots.find(
      (slot) => slot.participantId === initialState.onStage.viewerParticipantId,
    )?.body ?? "";
  const [onStage, setOnStage] = useState(initialState.onStage);
  const [onStageDraft, setOnStageDraft] = useState(initialOnStageDraft);
  const [backstage, setBackstage] = useState(initialState.backstage);
  const [backstageDraft, setBackstageDraft] = useState("");
  const [draft, setDraft] = useState("");
  const {
    inspector,
    close,
    openForCharacter,
    openForParticipant,
    setActiveCharacter,
  } = usePlayerHUDCharacterInspector();

  function controllerForCharacter(characterId: string) {
    return initialState.campaignNavigation.characterControllers.find((controller) =>
      controller.characters.some((character) => character.id === characterId),
    );
  }

  function nextResumeState(
    mode: PlayerHUDState["backstage"]["mode"],
    participants: BackstageParticipant[],
  ): PlayerHUDState["backstage"]["resumeState"] {
    if (mode !== "open") {
      return "inactive";
    }
    const players = participants.filter((participant) => participant.role === "player");
    return players.length > 0 && players.every((participant) => participant.readyToResume)
      ? "waiting-on-gm"
      : "collecting-ready";
  }

  function handleBackstageReadyToggle() {
    setBackstage((current) => {
      const participants = current.participants.map((participant) =>
        participant.id === current.viewerParticipantId
          ? { ...participant, readyToResume: !participant.readyToResume }
          : participant,
      );
      return {
        ...current,
        participants,
        resumeState: nextResumeState(current.mode, participants),
      };
    });
  }

  function handleBackstageSend() {
    const body = backstageDraft.trim();
    if (backstage.mode !== "open" || body.length === 0) {
      return;
    }
    setBackstage((current) => ({
      ...current,
      messages: [
        ...current.messages,
        {
          id: `shell-preview-${current.messages.length + 1}`,
          participantId: current.viewerParticipantId,
          body,
          sentAt: new Date().toISOString(),
        },
      ],
    }));
    setBackstageDraft("");
  }

  function handleOnStageWrite(yielded: boolean, nextMode: OnStageState["mode"]) {
    const body = onStageDraft.trim();
    if (body.length === 0 && !yielded) {
      return;
    }

    setOnStage((current) => {
      const nextSlots = [...current.slots];
      const index = nextSlots.findIndex(
        (slot) => slot.participantId === current.viewerParticipantId,
      );
      const existing = index >= 0 ? nextSlots[index] : undefined;
      const participant = current.participants.find(
        (entry) => entry.id === current.viewerParticipantId,
      );
      const nextSlot = {
        id: existing?.id ?? `${current.viewerParticipantId}-shell-preview`,
        participantId: current.viewerParticipantId,
        characters:
          existing?.characters.length
            ? existing.characters
            : participant?.characters ?? [],
        body: body || existing?.body,
        updatedAt: new Date().toISOString(),
        yielded,
        reviewState: yielded ? "under-review" : "open",
      } as const;

      if (index >= 0) {
        nextSlots[index] = nextSlot;
      } else {
        nextSlots.unshift(nextSlot);
      }

      return {
        ...current,
        mode: nextMode,
        slots: nextSlots,
        participants: current.participants.map((entry) =>
          entry.id === current.viewerParticipantId
            ? {
                ...entry,
                railStatus: yielded
                  ? "yielded"
                  : nextMode === "changes-requested"
                    ? "changes-requested"
                    : "active",
              }
            : entry,
        ),
        viewerControls: {
          ...current.viewerControls,
          canSubmit: !yielded && nextMode === "acting",
          canSubmitAndYield: !yielded && (nextMode === "acting" || nextMode === "changes-requested"),
          canYield: !yielded && nextMode === "acting",
          canUnyield: yielded,
          disabledReason: yielded
            ? "You have already yielded. Unyield if you need to revise before the beat closes."
            : current.viewerControls.disabledReason,
        },
      };
    });
  }

  function participantForActiveTab(participantId: string) {
    if (activeTab === "on-stage") {
      return onStage.participants.find((participant) => participant.id === participantId);
    }
    if (activeTab === "backstage") {
      return backstage.participants.find((participant) => participant.id === participantId);
    }
    return initialState.sideChat.participants.find(
      (participant) => participant.id === participantId,
    );
  }

  return (
    <>
      <PlayerHUDShell
        activeTab={activeTab}
        campaignNavigation={initialState.campaignNavigation}
        isSidebarOpen={isSidebarOpen}
        onSidebarOpenChange={setIsSidebarOpen}
        onTabChange={setActiveTab}
        onStage={onStage}
        onStageDraft={onStageDraft}
        onOnStageDraftChange={setOnStageDraft}
        onOnStageSubmit={() => handleOnStageWrite(false, "acting")}
        onOnStageSubmitAndYield={() => handleOnStageWrite(true, "yielded-waiting")}
        onOnStageYield={() => handleOnStageWrite(true, "yielded-waiting")}
        onOnStageUnyield={() =>
          setOnStage((current) => ({
            ...current,
            mode: "acting",
            slots: current.slots.map((slot) =>
              slot.participantId === current.viewerParticipantId
                ? { ...slot, yielded: false, reviewState: "open" }
                : slot,
            ),
            participants: current.participants.map((entry) =>
              entry.id === current.viewerParticipantId
                ? { ...entry, railStatus: "active" }
                : entry,
            ),
            viewerControls: {
              ...current.viewerControls,
              canSubmit: true,
              canSubmitAndYield: true,
              canYield: true,
              canUnyield: false,
              disabledReason: undefined,
            },
          }))
        }
        onCharacterInspect={(participantId, characterId) => {
          const controller =
            initialState.campaignNavigation.characterControllers.find(
              (entry) => entry.participantId === participantId,
            ) ??
            controllerForCharacter(characterId);
          if (!controller) {
            return;
          }
          openForCharacter(
            {
              name: controller.participantName,
              characters: controller.characters,
              isViewer: controller.isViewer,
            },
            characterId,
          );
        }}
        onParticipantInspect={(participantId) => {
          const participant = participantForActiveTab(participantId);
          if (!participant) {
            return;
          }
          openForParticipant({
            name: participant.name,
            characters: participant.characters,
            isViewer:
              activeTab === "on-stage"
                ? participant.id === onStage.viewerParticipantId
                : activeTab === "backstage"
                  ? participant.id === backstage.viewerParticipantId
                  : participant.id === initialState.sideChat.viewerParticipantId,
          });
        }}
        backstage={backstage}
        backstageDraft={backstageDraft}
        onBackstageDraftChange={setBackstageDraft}
        onBackstageSend={handleBackstageSend}
        onBackstageReadyToggle={handleBackstageReadyToggle}
        sideChat={initialState.sideChat}
        sideChatDraft={draft}
        onSideChatDraftChange={setDraft}
        onSideChatSend={() => setDraft("")}
      />
      <PlayerHUDCharacterInspectorDialog
        isOpen={Boolean(inspector)}
        participantName={inspector?.participantName ?? ""}
        characters={inspector?.characters ?? []}
        activeCharacterId={inspector?.activeCharacterId}
        isViewer={inspector?.isViewer ?? false}
        characterInspectionCatalog={initialState.campaignNavigation.characterInspectionCatalog}
        onCharacterChange={setActiveCharacter}
        onClose={close}
      />
    </>
  );
}
