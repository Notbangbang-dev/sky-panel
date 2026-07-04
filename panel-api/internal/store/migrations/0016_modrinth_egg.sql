-- +goose Up
-- A Minecraft egg that auto-installs a Modrinth modpack. itzg/minecraft-server
-- downloads and sets up the whole pack from the MODRINTH_MODPACK slug/URL when
-- TYPE=MODRINTH — so an empty startup (image entrypoint) is all that's needed.
INSERT INTO eggs (id, name, category, description, docker_image, startup, stop_command, variables_json, created_at)
VALUES (
    'e9b6c0de-4d21-4c8f-9d2a-1f0b7a3c5e10',
    'Modrinth Modpack',
    'Minecraft',
    'Auto-installs a modpack straight from Modrinth. Put a modpack slug or URL (e.g. "cobblemon" or a modrinth.com/modpack/... link) in the Modpack field; the loader and Minecraft version are pulled from the pack automatically.',
    'itzg/minecraft-server',
    '',
    'stop',
    '[{"name":"Modpack (slug or URL)","env":"MODRINTH_MODPACK","default":"","user_editable":true},{"name":"Minecraft Version","env":"VERSION","default":"","user_editable":true},{"name":"Release Channel","env":"MODRINTH_MODPACK_VERSION_TYPE","default":"release","user_editable":true},{"name":"Memory","env":"MEMORY","default":"3072M","user_editable":true},{"name":"EULA","env":"EULA","default":"TRUE","user_editable":true},{"name":"Type","env":"TYPE","default":"MODRINTH","user_editable":false}]',
    CURRENT_TIMESTAMP
);

-- +goose Down
DELETE FROM eggs WHERE id = 'e9b6c0de-4d21-4c8f-9d2a-1f0b7a3c5e10';
