-- name: GetCampaign :one
SELECT
	c.id, c.name, c.locale, c.game_system, c.status, c.gm_mode, c.intent, c.access_policy,
	-- TODO(session-readiness): include system-specific readiness in projection
	-- once system contracts are represented in projection state.
	CASE
		WHEN c.status NOT IN ('DRAFT', 'ACTIVE') THEN 0
		WHEN EXISTS (
			SELECT 1
			FROM sessions s
			WHERE s.campaign_id = c.id
				AND s.status = 'ACTIVE'
		) THEN 0
		WHEN NOT EXISTS (
			SELECT 1
			FROM participants p
			WHERE p.campaign_id = c.id
				AND p.role = 'GM'
		) THEN 0
		WHEN NOT EXISTS (
			SELECT 1
			FROM participants p
			WHERE p.campaign_id = c.id
				AND p.role = 'PLAYER'
		) THEN 0
		WHEN EXISTS (
			SELECT 1
			FROM characters ch
			WHERE ch.campaign_id = c.id
				AND TRIM(COALESCE(ch.controller_participant_id, '')) = ''
		) THEN 0
		WHEN EXISTS (
			SELECT 1
			FROM participants p
			WHERE p.campaign_id = c.id
				AND p.role = 'PLAYER'
				AND NOT EXISTS (
					SELECT 1
					FROM characters ch
					WHERE ch.campaign_id = c.id
						AND ch.controller_participant_id = p.id
				)
		) THEN 0
		WHEN c.game_system = 'DAGGERHEART'
				AND EXISTS (
					SELECT 1
					FROM characters ch
					LEFT JOIN daggerheart_character_profiles dcp
						ON ch.campaign_id = dcp.campaign_id AND ch.id = dcp.character_id
					WHERE ch.campaign_id = c.id
						AND (
							dcp.character_id IS NULL
							OR TRIM(COALESCE(dcp.class_id, '')) = ''
							OR TRIM(COALESCE(dcp.subclass_id, '')) = ''
							OR TRIM(COALESCE(dcp.ancestry_id, '')) = ''
							OR TRIM(COALESCE(dcp.community_id, '')) = ''
							OR COALESCE(dcp.traits_assigned, 0) = 0
							OR COALESCE(dcp.details_recorded, 0) = 0
							OR COALESCE(json_array_length(dcp.starting_weapon_ids_json), 0) = 0
							OR TRIM(COALESCE(dcp.starting_armor_id, '')) = ''
							OR TRIM(COALESCE(dcp.starting_potion_item_id, '')) = ''
							OR TRIM(COALESCE(dcp.background, '')) = ''
							OR COALESCE(json_array_length(dcp.experiences_json), 0) = 0
							OR COALESCE(json_array_length(dcp.domain_card_ids_json), 0) = 0
							OR TRIM(COALESCE(dcp.connections, '')) = ''
						)
				) THEN 0
		ELSE 1
	END AS can_start_session,
	(SELECT COUNT(*) FROM participants p WHERE p.campaign_id = c.id) AS participant_count,
	(SELECT COUNT(*) FROM characters ch WHERE ch.campaign_id = c.id) AS character_count,
	c.theme_prompt, c.cover_asset_id, c.cover_set_id, c.parent_campaign_id, c.fork_event_seq, c.origin_campaign_id,
	c.created_at, c.updated_at, c.completed_at, c.archived_at
FROM campaigns c WHERE c.id = ?;

-- name: PutCampaign :exec
INSERT INTO campaigns (
	id, name, locale, game_system, status, gm_mode, intent, access_policy,
	participant_count, character_count, theme_prompt, cover_asset_id, cover_set_id,
	created_at, updated_at, completed_at, archived_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	name = excluded.name,
	locale = excluded.locale,
	game_system = excluded.game_system,
	status = excluded.status,
	gm_mode = excluded.gm_mode,
	intent = excluded.intent,
	access_policy = excluded.access_policy,
    participant_count = excluded.participant_count,
    character_count = excluded.character_count,
    theme_prompt = excluded.theme_prompt,
    cover_asset_id = excluded.cover_asset_id,
    cover_set_id = excluded.cover_set_id,
    updated_at = excluded.updated_at,
    completed_at = excluded.completed_at,
    archived_at = excluded.archived_at;

-- name: ListCampaigns :many
SELECT
	c.id, c.name, c.locale, c.game_system, c.status, c.gm_mode, c.intent, c.access_policy,
	CASE
		WHEN c.status NOT IN ('DRAFT', 'ACTIVE') THEN 0
		WHEN EXISTS (
			SELECT 1
			FROM sessions s
			WHERE s.campaign_id = c.id
				AND s.status = 'ACTIVE'
		) THEN 0
		WHEN NOT EXISTS (
			SELECT 1
			FROM participants p
			WHERE p.campaign_id = c.id
				AND p.role = 'GM'
		) THEN 0
		WHEN NOT EXISTS (
			SELECT 1
			FROM participants p
			WHERE p.campaign_id = c.id
				AND p.role = 'PLAYER'
		) THEN 0
		WHEN EXISTS (
			SELECT 1
			FROM characters ch
			WHERE ch.campaign_id = c.id
				AND TRIM(COALESCE(ch.controller_participant_id, '')) = ''
		) THEN 0
		WHEN EXISTS (
			SELECT 1
			FROM participants p
			WHERE p.campaign_id = c.id
				AND p.role = 'PLAYER'
				AND NOT EXISTS (
					SELECT 1
					FROM characters ch
					WHERE ch.campaign_id = c.id
						AND ch.controller_participant_id = p.id
				)
		) THEN 0
		WHEN c.game_system = 'DAGGERHEART'
				AND EXISTS (
					SELECT 1
					FROM characters ch
					LEFT JOIN daggerheart_character_profiles dcp
						ON ch.campaign_id = dcp.campaign_id AND ch.id = dcp.character_id
					WHERE ch.campaign_id = c.id
						AND (
							dcp.character_id IS NULL
							OR TRIM(COALESCE(dcp.class_id, '')) = ''
							OR TRIM(COALESCE(dcp.subclass_id, '')) = ''
							OR TRIM(COALESCE(dcp.ancestry_id, '')) = ''
							OR TRIM(COALESCE(dcp.community_id, '')) = ''
							OR COALESCE(dcp.traits_assigned, 0) = 0
							OR COALESCE(dcp.details_recorded, 0) = 0
							OR COALESCE(json_array_length(dcp.starting_weapon_ids_json), 0) = 0
							OR TRIM(COALESCE(dcp.starting_armor_id, '')) = ''
							OR TRIM(COALESCE(dcp.starting_potion_item_id, '')) = ''
							OR TRIM(COALESCE(dcp.background, '')) = ''
							OR COALESCE(json_array_length(dcp.experiences_json), 0) = 0
							OR COALESCE(json_array_length(dcp.domain_card_ids_json), 0) = 0
							OR TRIM(COALESCE(dcp.connections, '')) = ''
						)
				) THEN 0
		ELSE 1
	END AS can_start_session,
	(SELECT COUNT(*) FROM participants p WHERE p.campaign_id = c.id) AS participant_count,
	(SELECT COUNT(*) FROM characters ch WHERE ch.campaign_id = c.id) AS character_count,
	c.theme_prompt, c.cover_asset_id, c.cover_set_id, c.parent_campaign_id, c.fork_event_seq, c.origin_campaign_id,
	c.created_at, c.updated_at, c.completed_at, c.archived_at
FROM campaigns c
WHERE c.id > ?
ORDER BY c.id
LIMIT ?;

-- name: ListAllCampaigns :many
SELECT
	c.id, c.name, c.locale, c.game_system, c.status, c.gm_mode, c.intent, c.access_policy,
	CASE
		WHEN c.status NOT IN ('DRAFT', 'ACTIVE') THEN 0
		WHEN EXISTS (
			SELECT 1
			FROM sessions s
			WHERE s.campaign_id = c.id
				AND s.status = 'ACTIVE'
		) THEN 0
		WHEN NOT EXISTS (
			SELECT 1
			FROM participants p
			WHERE p.campaign_id = c.id
				AND p.role = 'GM'
		) THEN 0
		WHEN NOT EXISTS (
			SELECT 1
			FROM participants p
			WHERE p.campaign_id = c.id
				AND p.role = 'PLAYER'
		) THEN 0
		WHEN EXISTS (
			SELECT 1
			FROM characters ch
			WHERE ch.campaign_id = c.id
				AND TRIM(COALESCE(ch.controller_participant_id, '')) = ''
		) THEN 0
		WHEN EXISTS (
			SELECT 1
			FROM participants p
			WHERE p.campaign_id = c.id
				AND p.role = 'PLAYER'
				AND NOT EXISTS (
					SELECT 1
					FROM characters ch
					WHERE ch.campaign_id = c.id
						AND ch.controller_participant_id = p.id
				)
		) THEN 0
		WHEN c.game_system = 'DAGGERHEART'
				AND EXISTS (
					SELECT 1
					FROM characters ch
					LEFT JOIN daggerheart_character_profiles dcp
						ON ch.campaign_id = dcp.campaign_id AND ch.id = dcp.character_id
					WHERE ch.campaign_id = c.id
						AND (
							dcp.character_id IS NULL
							OR TRIM(COALESCE(dcp.class_id, '')) = ''
							OR TRIM(COALESCE(dcp.subclass_id, '')) = ''
							OR TRIM(COALESCE(dcp.ancestry_id, '')) = ''
							OR TRIM(COALESCE(dcp.community_id, '')) = ''
							OR COALESCE(dcp.traits_assigned, 0) = 0
							OR COALESCE(dcp.details_recorded, 0) = 0
							OR COALESCE(json_array_length(dcp.starting_weapon_ids_json), 0) = 0
							OR TRIM(COALESCE(dcp.starting_armor_id, '')) = ''
							OR TRIM(COALESCE(dcp.starting_potion_item_id, '')) = ''
							OR TRIM(COALESCE(dcp.background, '')) = ''
							OR COALESCE(json_array_length(dcp.experiences_json), 0) = 0
							OR COALESCE(json_array_length(dcp.domain_card_ids_json), 0) = 0
							OR TRIM(COALESCE(dcp.connections, '')) = ''
						)
				) THEN 0
		ELSE 1
	END AS can_start_session,
	(SELECT COUNT(*) FROM participants p WHERE p.campaign_id = c.id) AS participant_count,
	(SELECT COUNT(*) FROM characters ch WHERE ch.campaign_id = c.id) AS character_count,
	c.theme_prompt, c.cover_asset_id, c.cover_set_id, c.parent_campaign_id, c.fork_event_seq, c.origin_campaign_id,
	c.created_at, c.updated_at, c.completed_at, c.archived_at
FROM campaigns c
ORDER BY c.id
LIMIT ?;
