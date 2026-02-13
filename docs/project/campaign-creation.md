# Campaign creation

## Creator identity
Campaign creation derives the creator participant display name from the auth user when `creator_display_name` is omitted. Callers should provide a valid user id (via metadata or tool input) so the game service can resolve the creator name from auth.
