-- +goose Up
-- Make the Node.js egg's entry file user-editable, the way Pterodactyl/cloud
-- panels do — so a server owner can point it at index.js, main.js, bot.js, etc.
-- from the Startup tab instead of being locked to `npm start`. The install
-- step is kept; we just run the chosen file directly. Existing Node servers
-- have no JS_FILE override, so they resolve to the default (index.js) on their
-- next re-provision — backwards compatible.
UPDATE eggs
SET startup = 'sh -c "npm install --omit=dev 2>/dev/null; node {{JS_FILE}}"',
    variables_json = '[{"name":"Main file","env":"JS_FILE","default":"index.js","user_editable":true},{"name":"Environment","env":"NODE_ENV","default":"production","user_editable":true}]'
WHERE id = '94ac77be-2f8c-4e39-82e8-0d970dd1c0d2';

-- +goose Down
UPDATE eggs
SET startup = 'sh -c "npm install --omit=dev 2>/dev/null; npm start"',
    variables_json = '[{"name":"Environment","env":"NODE_ENV","default":"production","user_editable":true}]'
WHERE id = '94ac77be-2f8c-4e39-82e8-0d970dd1c0d2';
