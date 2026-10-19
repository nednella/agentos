# Changelog

## [0.39.4](https://github.com/nednella/agentos/compare/v0.39.3...v0.39.4) (2026-10-19)


### Bug Fixes

* **ui:** scroll queue while issue is dragged near its edge ([53742fa](https://github.com/nednella/agentos/commit/53742fa4e6044759a4063fab494a8385ae1998ba))

## [0.39.3](https://github.com/nednella/agentos/compare/v0.39.2...v0.39.3) (2026-10-19)


### Bug Fixes

* **desktop:** show PR of resumed session at once ([506c513](https://github.com/nednella/agentos/commit/506c513d82b907dafd30173638a861062528b6b0))

## [0.39.2](https://github.com/nednella/agentos/compare/v0.39.1...v0.39.2) (2026-10-18)


### Bug Fixes

* **desktop:** tell no answer from an API error in gh and git calls ([6aab606](https://github.com/nednella/agentos/commit/6aab606da8a509ec9f0dfec564e4133db52bc49a))
* **internal:** read latest release from github.com redirect ([08f7b9d](https://github.com/nednella/agentos/commit/08f7b9d7c19cb3da26497d9f8f17d072bf2f3e08))
* **ui:** drop GitHub probe and countdown from offline splash ([339f7eb](https://github.com/nednella/agentos/commit/339f7ebb6c93450a79bbc0ea33dac580a6323fb6))
* **ui:** label offline splash button Retry ([0a258a1](https://github.com/nednella/agentos/commit/0a258a1f54c136c3fb0f111271f64f01d598f7ad))

## [0.39.1](https://github.com/nednella/agentos/compare/v0.39.0...v0.39.1) (2026-10-17)


### Bug Fixes

* **build:** sign releases with stable certificate ([544e81c](https://github.com/nednella/agentos/commit/544e81ca12df7dbbaa4161d0a48d902632921458))

## [0.39.0](https://github.com/nednella/agentos/compare/v0.38.1...v0.39.0) (2026-10-17)


### Features

* **cli:** add --chat flag to new ([25c7925](https://github.com/nednella/agentos/commit/25c7925ea5980a0a5afffec17742ab57b1f79225))
* **desktop:** start chat from new command when asked ([e53b91a](https://github.com/nednella/agentos/commit/e53b91ac2788f09a8d8e6f1150e385dabc81c78b))
* **internal:** name agentos new --chat in session prompt ([8aaace4](https://github.com/nednella/agentos/commit/8aaace429ca172757e47efd0bf830490bf4200c9))

## [0.38.1](https://github.com/nednella/agentos/compare/v0.38.0...v0.38.1) (2026-10-17)


### Bug Fixes

* **desktop:** save session identity at start and rename ([dd005f3](https://github.com/nednella/agentos/commit/dd005f3cb5d8433fd64124806dd361f13d433c1d))
* **internal:** keep session identity through hooks ([a9645b5](https://github.com/nednella/agentos/commit/a9645b50521dc613260711b424a06e2a18345f60))

## [0.38.0](https://github.com/nednella/agentos/compare/v0.37.0...v0.38.0) (2026-10-16)


### Features

* **desktop:** emit network event when GitHub or git call fails ([efe448e](https://github.com/nednella/agentos/commit/efe448e4481669f8204ae7e8332dc9d22ed2acda))


### Bug Fixes

* **ui:** check network when app reports failed call ([1c92e13](https://github.com/nednella/agentos/commit/1c92e13d0e6a8c4a6fdc0760e81ce4a1378101c1))

## [0.37.0](https://github.com/nednella/agentos/compare/v0.36.1...v0.37.0) (2026-10-16)


### Features

* **ui:** show splash while network is unavailable ([1e80f13](https://github.com/nednella/agentos/commit/1e80f13a4e154fe0b63e664fcf9d569e5736f4d0)), closes [#436](https://github.com/nednella/agentos/issues/436)
* **ui:** stack offline countdown above retry button ([c01ff00](https://github.com/nednella/agentos/commit/c01ff00f3014b536d8b2f14b73835ddfa0da77b6))

## [0.36.1](https://github.com/nednella/agentos/compare/v0.36.0...v0.36.1) (2026-10-16)


### Bug Fixes

* **desktop:** hold page calls until the page is attached again ([fde4119](https://github.com/nednella/agentos/commit/fde4119c902faaccc95544e380e53318bbf7fe43))
* **desktop:** reconnect to a browser whose connection dropped ([83d1dd5](https://github.com/nednella/agentos/commit/83d1dd509fee75ac4707876baf2352c2ecf600ca)), closes [#434](https://github.com/nednella/agentos/issues/434)
* **ui:** size stat project dropdown to its choice ([8a93c5e](https://github.com/nednella/agentos/commit/8a93c5e6f0b776a33d5370112288939b6938d595))

## [0.36.0](https://github.com/nednella/agentos/compare/v0.35.0...v0.36.0) (2026-10-15)


### Features

* **ui:** label top bar memory total as mem ([36f4b12](https://github.com/nednella/agentos/commit/36f4b12210960e047d4c3408e380871f175baa4a))
* **ui:** show memory total in top bar ([9d09d8c](https://github.com/nednella/agentos/commit/9d09d8c71e6c1113e1044e9b8d78af5ff04e08f4))


### Bug Fixes

* **ui:** separate memory from cost in session row ([d1cb626](https://github.com/nednella/agentos/commit/d1cb626bd5deba7681aa871fbbe896317d41b640)), closes [#426](https://github.com/nednella/agentos/issues/426)

## [0.35.0](https://github.com/nednella/agentos/compare/v0.34.7...v0.35.0) (2026-10-15)


### Features

* **desktop:** list running browser processes ([f527df7](https://github.com/nednella/agentos/commit/f527df77abf5dd5c3453f97d2b1b9fafd425fed1))
* **desktop:** resume ended session by hand ([27e8b09](https://github.com/nednella/agentos/commit/27e8b09ebd287430765d42f764dc195fa95957ac))
* **desktop:** sample memory of each session and of agentos ([36eff7e](https://github.com/nednella/agentos/commit/36eff7e34d970dd77d9864da3112dc3baf4593be))
* **internal:** read session pane process and tmux server process ([cd190f5](https://github.com/nednella/agentos/commit/cd190f5e5454cefaf78dd7957206e4f64962ca35))
* **internal:** start browser MCP server at first tool call ([4751acb](https://github.com/nednella/agentos/commit/4751acbb59a77410ca836de09b7d36fbec241c2f)), closes [#417](https://github.com/nednella/agentos/issues/417)
* **ui:** show resume on ended rows and in palette ([0ab3e15](https://github.com/nednella/agentos/commit/0ab3e1569874e7beb3b5b5d89ad08665fc7bc0a0))
* **ui:** show session memory and agentos total ([c6769a4](https://github.com/nednella/agentos/commit/c6769a4ddc80b01835d74049cd6ca328d46b0a23))


### Bug Fixes

* **desktop:** count compressed memory in session figures ([a004f36](https://github.com/nednella/agentos/commit/a004f36028ccfe5b8086f4d3867338141f54a3a9))
* **desktop:** give Claude its sent prompt as start argument ([b2b6edf](https://github.com/nednella/agentos/commit/b2b6edf7726f1c104ee5fa1ae4fa19bb7b42cf57)), closes [#418](https://github.com/nednella/agentos/issues/418)
* **desktop:** stop app's watchers and commands when it quits ([93ca710](https://github.com/nednella/agentos/commit/93ca7100f8642702dfc70b59ca48e82722d3ba3a)), closes [#417](https://github.com/nednella/agentos/issues/417)
* **desktop:** stop browser when its first window cannot open ([599be58](https://github.com/nednella/agentos/commit/599be58e825a25eeab6afa74936154bc45e3deab)), closes [#417](https://github.com/nednella/agentos/issues/417)
* **internal:** stop every process under killed session's pane ([e9ff49a](https://github.com/nednella/agentos/commit/e9ff49af86cb2370148eaf0768015208788cf36a)), closes [#417](https://github.com/nednella/agentos/issues/417)
* **internal:** tell reused pid apart by its start time ([f9f3b76](https://github.com/nednella/agentos/commit/f9f3b768e93ad4494e6a12412dc08ddfb2dc8128)), closes [#417](https://github.com/nednella/agentos/issues/417)
* **ui:** hide queue row's model tag when title has no room ([29f4074](https://github.com/nednella/agentos/commit/29f40747d40b59b175fc7f2402d4c26c91431d74))
* **ui:** keep type mark from shrinking in narrow rows ([a0333fe](https://github.com/nednella/agentos/commit/a0333fe4dcbef1d5a8f6bbbcb83a429cfb0cb00e))

## [0.34.7](https://github.com/nednella/agentos/compare/v0.34.6...v0.34.7) (2026-10-15)


### Bug Fixes

* **ui:** keep news date header above scrolled rows ([4821029](https://github.com/nednella/agentos/commit/48210290742e25a6a6ea90c119857c3c60fa744b)), closes [#412](https://github.com/nednella/agentos/issues/412)

## [0.34.6](https://github.com/nednella/agentos/compare/v0.34.5...v0.34.6) (2026-10-14)


### Bug Fixes

* **internal:** let chat browser tools that change the page run without asking ([15a62f3](https://github.com/nednella/agentos/commit/15a62f3001cc3ac9f04b9eb1e9cc34e0bea66bca))
* **internal:** let chats run agentos browser without asking ([2c10d77](https://github.com/nednella/agentos/commit/2c10d7714d724a7746dab7b13c197515cdc80433)), closes [#407](https://github.com/nednella/agentos/issues/407)

## [0.34.5](https://github.com/nednella/agentos/compare/v0.34.4...v0.34.5) (2026-10-14)


### Bug Fixes

* **ui:** show focus only on panel that holds it ([9e9cb4a](https://github.com/nednella/agentos/commit/9e9cb4ad9bb3e7bbcdd03cf5bb6afd31a76b3c7d))

## [0.34.4](https://github.com/nednella/agentos/compare/v0.34.3...v0.34.4) (2026-10-14)


### Bug Fixes

* **desktop:** let first click on inactive window reach page ([99c928d](https://github.com/nednella/agentos/commit/99c928d78ad3ccc433d81abaf5917bf12faa6500))

## [0.34.3](https://github.com/nednella/agentos/compare/v0.34.2...v0.34.3) (2026-10-14)


### Bug Fixes

* **desktop:** ignore PR that closed before session started ([757ea15](https://github.com/nednella/agentos/commit/757ea15b49e4b65f74b0a6d08c413609029c3d4b))

## [0.34.2](https://github.com/nednella/agentos/compare/v0.34.1...v0.34.2) (2026-10-14)


### Bug Fixes

* **desktop:** do not count approvals as reviews to address ([34b6c37](https://github.com/nednella/agentos/commit/34b6c37db71b7a790a406c3710c11bc501d91ad4))
* **desktop:** send no review command while the PR is a draft ([36e3e46](https://github.com/nednella/agentos/commit/36e3e46eab8d5d7a51475c82f052d60eb2859fdf)), closes [#392](https://github.com/nednella/agentos/issues/392)
* **desktop:** track project folder branch only when session switched to it ([fed877d](https://github.com/nednella/agentos/commit/fed877d9f7a13a085e4632151c408b50a81e5ef3)), closes [#393](https://github.com/nednella/agentos/issues/393)

## [0.34.1](https://github.com/nednella/agentos/compare/v0.34.0...v0.34.1) (2026-10-13)


### Bug Fixes

* **ui:** dim heat map days outside chosen range ([1d1414b](https://github.com/nednella/agentos/commit/1d1414bb0d0460152e41a58e4b5929e71b3ed661))

## [0.34.0](https://github.com/nednella/agentos/compare/v0.33.0...v0.34.0) (2026-10-13)


### Features

* **desktop:** keep session's action apart from its title ([8ccacf9](https://github.com/nednella/agentos/commit/8ccacf9a72c84cf8c3afa419fe1a496dd96c1fd0))
* **ui:** add Today, 7D, 90D and 180D stat ranges ([e74e3b4](https://github.com/nednella/agentos/commit/e74e3b41057ee41ebc5f24d47d236a2b3115ff75))
* **ui:** pick stat range from dropdown ([4d1dc20](https://github.com/nednella/agentos/commit/4d1dc2008921779ecf436f9826ed9b9113bab6b9))
* **ui:** show session's action as badge ([8bdc855](https://github.com/nednella/agentos/commit/8bdc855030a1dcc048defea0e64d709ea965daee))


### Bug Fixes

* **desktop:** drive Chrome for Testing, never the owner's browser ([cdb2980](https://github.com/nednella/agentos/commit/cdb29801a0f520e8773b6b5b3c54454350d22167))
* **scripts:** drive Chrome for Testing in cdp.mjs ([fc7ea00](https://github.com/nednella/agentos/commit/fc7ea005e5d69839d0bf78b8bc216cfc0764311e))
* **ui:** join issue row actions into one split button ([cd9a2b7](https://github.com/nednella/agentos/commit/cd9a2b7ab5b4057590eefd02c1a49ce406f7f6ed))
* **ui:** size stat range dropdown to its choice ([f0d1033](https://github.com/nednella/agentos/commit/f0d1033ff51362902a5632a1df0a09cd3d487f14))

## [0.33.0](https://github.com/nednella/agentos/compare/v0.32.0...v0.33.0) (2026-10-13)


### Features

* **desktop:** start chats with browser ([aa3801e](https://github.com/nednella/agentos/commit/aa3801eff468796df8920c2e5cc02ee9b10e855d))
* **internal:** let chat with browser use page tools in plan mode ([e07726e](https://github.com/nednella/agentos/commit/e07726ed2d7731f4d59cf03eb568436c2336eb33))
* **internal:** show session whose reply ends in question as waiting ([a63a8d4](https://github.com/nednella/agentos/commit/a63a8d45610f03271d67c1253f45c6c6950713b1))
* **internal:** tell sessions to ask the owner with AskUserQuestion ([9292b5c](https://github.com/nednella/agentos/commit/9292b5c2d67f95d1d36923423bdd0982e9c81cea))
* **ui:** show issue's chat beside its task in queue ([2f276ee](https://github.com/nednella/agentos/commit/2f276eea94947380e629b639f1c1f55b172ecab5))
* **ui:** show terminal, browser and evidence tabs for chats ([7f2844b](https://github.com/nednella/agentos/commit/7f2844bd3c7609a44f5afc40eed0f588197f6d9f))


### Bug Fixes

* **ui:** link queue row to issue's task, not its chat ([64c72d3](https://github.com/nednella/agentos/commit/64c72d39f4f2f6487f430394308d15c3d95cc2cc))

## [0.32.0](https://github.com/nednella/agentos/compare/v0.31.0...v0.32.0) (2026-10-12)


### Features

* **desktop:** name action in title of issue session ([aeb87e4](https://github.com/nednella/agentos/commit/aeb87e4794c3f589a6290d8b0d76988c79388748))
* **desktop:** start chat for action of kind chat ([1b580fb](https://github.com/nednella/agentos/commit/1b580fbc7ea195a944ce7602b5790835ecd395b5))
* **internal:** let chat comment on issues and change their labels ([1a2f46c](https://github.com/nednella/agentos/commit/1a2f46c4cb7fd2b5424b312e873b31ae324620a0))
* **internal:** let queue action set kind of session ([960a71b](https://github.com/nednella/agentos/commit/960a71be3f316ca64e7de49374b4b99f090bcad7))
* **internal:** tell sessions to refresh queue after label change ([0a844c0](https://github.com/nednella/agentos/commit/0a844c0e1ad9354f024d3d65371e6c585be7969d))

## [0.31.0](https://github.com/nednella/agentos/compare/v0.30.1...v0.31.0) (2026-10-12)


### Features

* **ui:** mark new TLDR stories until opened ([5650f75](https://github.com/nednella/agentos/commit/5650f75e73b8aa7a6e672a133409fc928355f846))


### Bug Fixes

* **desktop:** tell web view its window is key once page loads ([62181b1](https://github.com/nednella/agentos/commit/62181b137a9fe15be3cef4426a6d8e26a0a91a4a))

## [0.30.1](https://github.com/nednella/agentos/compare/v0.30.0...v0.30.1) (2026-10-12)


### Bug Fixes

* **cli:** limit --prompt to 6 KB ([cb110f0](https://github.com/nednella/agentos/commit/cb110f058cfe856b416500199021afbf77a3a0b1))
* **desktop:** block clean-up of worktree in the middle of a rebase ([bb54ac7](https://github.com/nednella/agentos/commit/bb54ac714ad4979694ba063bea0ff5ed48ed9476))
* **desktop:** clean up only branches a session owns ([c0bce2a](https://github.com/nednella/agentos/commit/c0bce2a685c40d38bb761f897192002698849988)), closes [#337](https://github.com/nednella/agentos/issues/337)
* **desktop:** count usage at each hook and record it when the turn ends ([6fbe3f1](https://github.com/nednella/agentos/commit/6fbe3f1e5fa3d538cef1b419ed982fac083d13b4))
* **desktop:** count usage while a session works ([8c36ad1](https://github.com/nednella/agentos/commit/8c36ad11216e41e49d26b0069eb2ad07c1ea0411))
* **desktop:** drop held PR command when its session ends ([c3846f3](https://github.com/nednella/agentos/commit/c3846f3ea0cfe58248fca56744c18dda896ffc07))
* **desktop:** follow session_prompt_send when resuming session ([e0e9e4b](https://github.com/nednella/agentos/commit/e0e9e4b097d9d3c453b57f68252b6d8a7af1921f))
* **desktop:** frame agentos new prompt from session as handover ([16cef8c](https://github.com/nednella/agentos/commit/16cef8c19f2304e1b78fac7f68f07fa395300418))
* **desktop:** keep a session's timeline and worked time across a restart ([5f84ad6](https://github.com/nednella/agentos/commit/5f84ad60c8e188062ee1676438cd98bd2637d7a1))
* **desktop:** keep both stats tabs loaded when switching ([ea1d8ee](https://github.com/nednella/agentos/commit/ea1d8eed388478c7140cc88e89f4a8a37697dac4))
* **desktop:** keep ended session's branches and handover on resume ([04663b0](https://github.com/nednella/agentos/commit/04663b0e10e70dcdc5b7cda5c228554b8f9b46ce))
* **desktop:** name issue's branch in start line ([e30226e](https://github.com/nednella/agentos/commit/e30226e64d4efc8cbb329179a2abfbd62a3a0fef))
* **desktop:** queue PR commands held while session waits ([fc5fde9](https://github.com/nednella/agentos/commit/fc5fde9e598bc4d39d23945b7ec57ef4b56dddcf))
* **desktop:** read issue list again for issue not in it ([27ddeb8](https://github.com/nednella/agentos/commit/27ddeb8e3af3aea4d9a73801f700a171f87a5181))
* **desktop:** refuse clean-up of branch the session no longer has ([4af95b4](https://github.com/nednella/agentos/commit/4af95b4da6618e6d0b486b7f905170d049571b06))
* **desktop:** refuse default branch in agentos track ([422b39a](https://github.com/nednella/agentos/commit/422b39a394f2ea124f19b95c960caad623f45c0c))
* **desktop:** send PR commands only to session that hears the PR ([5b4f554](https://github.com/nednella/agentos/commit/5b4f554cc32c2a7dda1a8b0a593dc0c9f70a2e15))
* **desktop:** skip resume when issue has live session ([2ffd5b6](https://github.com/nednella/agentos/commit/2ffd5b69d5f030933b25a68caca2e69ce22aa2e7))
* **desktop:** start one session per issue at a time ([0a6fe6a](https://github.com/nednella/agentos/commit/0a6fe6a52623929e4f4433fb209cd45d7dec7527))
* **internal:** ask owner before setup session acts on PR command ([93c4f77](https://github.com/nednella/agentos/commit/93c4f77b1ac95f73e8bbf342f299a4538cb158ca))
* **internal:** give tasks one way to hand work on ([7760d92](https://github.com/nednella/agentos/commit/7760d928f6f5cf4151a41b0244dfaa5c9803bc3e))
* **internal:** let chat file issue and start its task ([2aa51dc](https://github.com/nednella/agentos/commit/2aa51dcf7c3e8ee13f681298cc0529194760aecb))

## [0.30.0](https://github.com/nednella/agentos/compare/v0.29.0...v0.30.0) (2026-10-11)


### Features

* **cli:** add --prompt to agentos issue ([f4d6a89](https://github.com/nednella/agentos/commit/f4d6a89f67bb12db6fb9e093e349a5af3521438c))
* **desktop:** count chat usage by model instead of in a Chats row ([42b3b9c](https://github.com/nednella/agentos/commit/42b3b9cf60ffbf9bd8dffda9d7f1dd94221726bb))
* **desktop:** count chat usage in total cost and its own row ([bcbeaa6](https://github.com/nednella/agentos/commit/bcbeaa622fa29218555738ee791d295759dc6442))
* **desktop:** give issue session handover from session that asked ([7d229b8](https://github.com/nednella/agentos/commit/7d229b85ba14613061690fc2becba8fc1af0c85f))
* **desktop:** record usage of chats as chat_usage events ([3d80d55](https://github.com/nednella/agentos/commit/3d80d559be717049979c228b7aec4268c6cd34ee))
* **internal:** tell tasks to start issue work with agentos issue ([2aba077](https://github.com/nednella/agentos/commit/2aba07764834100ed7dfae6ed3333c6147f2b7cc)), closes [#332](https://github.com/nednella/agentos/issues/332)
* **ui:** move to line start and end with Cmd+Left and Cmd+Right in the terminal ([70019f0](https://github.com/nednella/agentos/commit/70019f06329cd3869f7fea580d7498d77a7245f9))
* **ui:** select and clear name before splash fades ([b449981](https://github.com/nednella/agentos/commit/b44998162918fa81ca9186b54ada04e749f4637d))
* **ui:** show chat cost in spend ([e838740](https://github.com/nednella/agentos/commit/e83874094963ce1192ea132957bf96a0b7ee8aef))
* **ui:** show chats in spend model rows as sessions ([d29c8aa](https://github.com/nednella/agentos/commit/d29c8aa062c10f1ebee185b19e0b223ab3ab2c6d))


### Bug Fixes

* **ui:** drop window button space in full screen ([e7a4723](https://github.com/nednella/agentos/commit/e7a47239cc15257e7cd42c3fa816a49bd2eade9d))
* **ui:** keep top bar 40px high at every text size ([78dd237](https://github.com/nednella/agentos/commit/78dd23769c8e4ccf22d1474d7cfa11eee7a23b2b))

## [0.29.0](https://github.com/nednella/agentos/compare/v0.28.0...v0.29.0) (2026-10-11)


### Features

* **desktop:** count chat starts and session time in ledger ([77bede1](https://github.com/nednella/agentos/commit/77bede1bfde7f415360e4d4d4bbfe0c2adfc952e))
* **desktop:** scope stats and ledger to a project ([d322445](https://github.com/nednella/agentos/commit/d32244508e3d85cd6fd551deefd825b11118127d))
* **ui:** add project picker to stats ([a67babb](https://github.com/nednella/agentos/commit/a67babb9721379b01e4a07f27d24dcd3186821b9))
* **ui:** cap stats tables with a show-all page ([17894af](https://github.com/nednella/agentos/commit/17894af50db3ff03ef1a2ab7c8468bd7840fd42b))
* **ui:** cap work table at 5 rows ([74f5132](https://github.com/nednella/agentos/commit/74f51327bf5c3b7271a35610afc86f4c61d8eebd))
* **ui:** drop last-active column from work table ([9f21a33](https://github.com/nednella/agentos/commit/9f21a33473378cba19c3f92b46f2901f5545ad27))
* **ui:** name stats cost columns as estimates ([6f20872](https://github.com/nednella/agentos/commit/6f2087256089246736c46dd7b246bae7d0ed68ea))
* **ui:** show chats and session time in stats ([d63d15c](https://github.com/nednella/agentos/commit/d63d15c42ab8300a49adff02724062183ce709ae))
* **ui:** split stats into sections with a rail ([019411c](https://github.com/nednella/agentos/commit/019411c1eb599179ed17395d7a276d873b0d360d)), closes [#327](https://github.com/nednella/agentos/issues/327)


### Bug Fixes

* **ui:** call stats cost columns Cost ([111b4f7](https://github.com/nednella/agentos/commit/111b4f7616c6a1523ec198942ac578f4d478b8ed))
* **ui:** give stats sections more room ([3ab65d8](https://github.com/nednella/agentos/commit/3ab65d81a8272b73d2e8ba69be75ca35ae423957))
* **ui:** lay stats top bar out in two rows when narrow ([49e7789](https://github.com/nednella/agentos/commit/49e7789630a3812dac7437d2d41471645c423e6b))
* **ui:** match stats text inset on both sides ([ff57e20](https://github.com/nednella/agentos/commit/ff57e204f10e214723c63c3f27195d3de070a141))
* **ui:** name current figures in empty stats ([4099b5d](https://github.com/nednella/agentos/commit/4099b5dd0b58da6aad96a371656861320c6509a4))
* **ui:** use 16px side padding on narrow stats panel ([8b2bf60](https://github.com/nednella/agentos/commit/8b2bf60d499f956383a8eff6a573e65ee4541d9b))

## [0.28.0](https://github.com/nednella/agentos/compare/v0.27.2...v0.28.0) (2026-10-11)


### Features

* **desktop:** record token usage of each turn ([8ce67ad](https://github.com/nednella/agentos/commit/8ce67ad683b51ef37038f8505f3fa3069cd58343))
* **desktop:** sum token usage in ledger ([9ad7317](https://github.com/nednella/agentos/commit/9ad7317f1b1dcb14b28427a3c0a1c2e876eef319))
* **internal:** keep transcript path from hook events ([f655973](https://github.com/nednella/agentos/commit/f6559734980e566d243e68d1cad2c8875a31b9a8))
* **internal:** read token usage and cost from Claude transcripts ([7e260e5](https://github.com/nednella/agentos/commit/7e260e549c500d281b64147320a977796aa312da))
* **ui:** draw loader as app icon prompt typing ([9eb1d90](https://github.com/nednella/agentos/commit/9eb1d90c7e99c9c69a5db0f3f7ce41efecc5b2c3))
* **ui:** draw loader at 22px ([f7d3858](https://github.com/nednella/agentos/commit/f7d38589205b4f81ef5bfce2e5dce9eff91900ed))
* **ui:** hold loader for a second on first load ([9291aa8](https://github.com/nednella/agentos/commit/9291aa8ece824329f072dbec11ace6d2f639308a))
* **ui:** show centred spinner while views load ([d2a8256](https://github.com/nednella/agentos/commit/d2a825652be2bfe494d024274d8a654c0a1a6b40)), closes [#325](https://github.com/nednella/agentos/issues/325)
* **ui:** show splash with icon's prompt when app opens ([3cc31c0](https://github.com/nednella/agentos/commit/3cc31c0f402d23ff62e46f4a11832e829cadefc4)), closes [#329](https://github.com/nednella/agentos/issues/329)
* **ui:** show tokens and cost in session header ([73fdc64](https://github.com/nednella/agentos/commit/73fdc64303d2a3873823facc72ebfadaadc57c79))
* **ui:** show tokens and estimated cost ([88b697b](https://github.com/nednella/agentos/commit/88b697bdab5b25f90c42a7c365206e0fc61eaff7)), closes [#320](https://github.com/nednella/agentos/issues/320)


### Bug Fixes

* **browser:** start Brave with a mock keychain ([e7fad58](https://github.com/nednella/agentos/commit/e7fad5815566d48c8b5590ee15fd2aad2d31aa80))
* **desktop:** leave branch another session works on to that session ([0ea05ad](https://github.com/nednella/agentos/commit/0ea05ada7bd7933d762731a494eab91290b239a6)), closes [#330](https://github.com/nednella/agentos/issues/330)
* **scripts:** stop Brave on every exit of cdp.mjs ([e66c557](https://github.com/nednella/agentos/commit/e66c557c01d32230ac4cf143e4e62377dbf52a51))
* **ui:** separate effort from spend on session row and header ([7f536ba](https://github.com/nednella/agentos/commit/7f536ba8f5dd89076bb341c4d553ce5ed196ee8b))
* **ui:** show model, effort and cost on session row ([5ce61e8](https://github.com/nednella/agentos/commit/5ce61e8098a2e896c927912f63798f6706a5e7e4))
* **ui:** show Thinking for busy chat ([99ece80](https://github.com/nednella/agentos/commit/99ece80e3590c458934ebcbbe01728b425e5757a)), closes [#319](https://github.com/nednella/agentos/issues/319)

## [0.27.2](https://github.com/nednella/agentos/compare/v0.27.1...v0.27.2) (2026-10-11)


### Bug Fixes

* **ui:** swap new chat and new task shortcuts ([d72bdd4](https://github.com/nednella/agentos/commit/d72bdd46e58cde60ad15ee4d399ef3ae4eb1e582)), closes [#316](https://github.com/nednella/agentos/issues/316)

## [0.27.1](https://github.com/nednella/agentos/compare/v0.27.0...v0.27.1) (2026-10-11)


### Bug Fixes

* **desktop:** stop counting finished turns as interruptions ([4adf56d](https://github.com/nednella/agentos/commit/4adf56dc2b31ec3977c1ae9418b522974da59033))

## [0.27.0](https://github.com/nednella/agentos/compare/v0.26.0...v0.27.0) (2026-10-11)


### Features

* **desktop:** free ⌘M from Minimize ([8ec3ff6](https://github.com/nednella/agentos/commit/8ec3ff694cc2286624d03db171f74e3737d33e97))
* **desktop:** start read-only chats in a project ([d754f81](https://github.com/nednella/agentos/commit/d754f819d0c9184272f244767b8479a1484d8575))
* **ui:** call working sessions tasks ([7e637ed](https://github.com/nednella/agentos/commit/7e637edbcfc7b28b32fc0e1e17bfe308651e9cbc))
* **ui:** offer chat and task cards on empty screen ([fd22686](https://github.com/nednella/agentos/commit/fd226864fc9103601f702362e6ffc350c3c11f31))
* **ui:** say agent where text means session or chat ([6c9c701](https://github.com/nednella/agentos/commit/6c9c70115b9e094f36669d5b5bed38c85765db66))
* **ui:** say no chats yet in empty chats group ([1e919ea](https://github.com/nednella/agentos/commit/1e919eaea983ec154e36ed058b247524586881ca))
* **ui:** show chat shortcut on empty screen and in sheet ([6469ee1](https://github.com/nednella/agentos/commit/6469ee1acf84c36cc851e6f061f1a66369a1e425))
* **ui:** show chats above sessions in agents panel ([2aece6f](https://github.com/nednella/agentos/commit/2aece6f2a7e3f28eac5bf4be34d3edfa1f6211a6))
* **ui:** show icon and hint in empty session groups ([b11e87d](https://github.com/nednella/agentos/commit/b11e87d5296cf9f5a6448085bb5d293c711d9c3b))

## [0.26.0](https://github.com/nednella/agentos/compare/v0.25.0...v0.26.0) (2026-10-10)


### Features

* **stats:** read interruptions from event log ([9565910](https://github.com/nednella/agentos/commit/9565910d013c6a034c0bc0670e41a75cb7f80b9a))
* **stats:** write event log per machine, re-read changed files ([05b2c20](https://github.com/nednella/agentos/commit/05b2c20e7a5bfd82d2f11d531113a92129abd50a))
* **ui:** add shortcut to collapse all panels ([bfe1fc4](https://github.com/nednella/agentos/commit/bfe1fc40220ba16c01c83a1c232979a3a53e1d25))
* **ui:** move stats and news shortcuts to option ([e722b25](https://github.com/nednella/agentos/commit/e722b25b8cc7b0b085b93449be55550191ecf560))
* **ui:** toggle all panels with cmd+b ([3516562](https://github.com/nednella/agentos/commit/35165628d78e32bc969eab8c4af25cd9de4ddb99))
* **ui:** toggle each panel with shift on its focus key ([7ab315b](https://github.com/nednella/agentos/commit/7ab315b98cec57658dca50b4835a3ebc968007cf))


### Bug Fixes

* **desktop:** replace machine id of wrong shape ([6bb9691](https://github.com/nednella/agentos/commit/6bb9691e43504aef850bec8843b5647b64237887))

## [0.25.0](https://github.com/nednella/agentos/compare/v0.24.0...v0.25.0) (2026-10-09)


### Features

* **desktop:** clean up each branch of a session as its PR finishes ([e364d19](https://github.com/nednella/agentos/commit/e364d19ffdda38394b08b446339d9a32e66170ff))
* **desktop:** give setup session only its brief ([11171a1](https://github.com/nednella/agentos/commit/11171a1e96768845db5723f3e6b104677372284b))
* **desktop:** name branch in clean-up block reasons ([740dd18](https://github.com/nednella/agentos/commit/740dd1812814ae3edabb520d10f5c20e9ee43712))
* **desktop:** pick up config edits while app runs ([82bf5c7](https://github.com/nednella/agentos/commit/82bf5c780eb7363a4ef6b684a24fbd70793cbe2e))
* **internal:** brief new users in set-up session ([20f685e](https://github.com/nednella/agentos/commit/20f685e82b83b1c1dcdd19ce4ae1f65fbc0019ab))
* **internal:** explain APPLICATION_PROMPT in setup brief ([164af15](https://github.com/nednella/agentos/commit/164af15e4d4c7c168da6bfde6a016d4d89a20737))
* **internal:** tell sessions to hand other topics to new session ([9518028](https://github.com/nednella/agentos/commit/9518028a3a47104d8c8a869b9514527307b3e4f7))


### Bug Fixes

* **desktop:** keep hand edits to config when app saves ([edecfe8](https://github.com/nednella/agentos/commit/edecfe874f931d4f285915cb2c752cfb75b6e26f))
* **internal:** stop hard-wrapping browser help ([4ac8801](https://github.com/nednella/agentos/commit/4ac88010c63591447c275cbed29169bcb9f73bcf))
* **internal:** unwrap work command template in setup brief ([aeebde4](https://github.com/nednella/agentos/commit/aeebde45bc1fe755177f4fa4e97ea8a7ebe223af))

## [0.24.0](https://github.com/nednella/agentos/compare/v0.23.0...v0.24.0) (2026-10-09)


### Features

* **desktop:** mark prompts the app types into a session ([0fd0475](https://github.com/nednella/agentos/commit/0fd0475f1928924879872d0d56e3608d12a941f9))
* **internal:** tell sessions how to branch and what the app sends ([b63c49e](https://github.com/nednella/agentos/commit/b63c49ee37a3724a81b84aa19642bb7e1b288ab4))
* **internal:** tell sessions to work on a branch ([4e6fdba](https://github.com/nednella/agentos/commit/4e6fdba0b2c6f335ea58406e23c1aabf1ee3e9fb))
* **ui:** keep close button in place on news pages ([d5b23a6](https://github.com/nednella/agentos/commit/d5b23a65d7058afd04e417f5a1c96000dabb3cf6))
* **ui:** show back icon beside News in news pages ([3f5a210](https://github.com/nednella/agentos/commit/3f5a210e0735953150f96422af190ce6b0f1a41c))


### Bug Fixes

* **ui:** use chevron for back icon beside News ([732a158](https://github.com/nednella/agentos/commit/732a15806f3c45fb11361cf1c7053b257b4728d6))

## [0.23.0](https://github.com/nednella/agentos/compare/v0.22.1...v0.23.0) (2026-10-09)


### Features

* **internal:** give set-up sessions fixed templates marked by agentos ([572629d](https://github.com/nednella/agentos/commit/572629d47f21d2b113fbc8e2704c37b46b94d786))


### Bug Fixes

* **internal:** fence dependency list and command in digest prompt ([a1cd43e](https://github.com/nednella/agentos/commit/a1cd43eea9315d801f738ec503611a5400a652c3))
* **ui:** match empty state description to README ([b0cbc1b](https://github.com/nednella/agentos/commit/b0cbc1b4a7471ec9f257172f5dd21618765eb199))

## [0.22.1](https://github.com/nednella/agentos/compare/v0.22.0...v0.22.1) (2026-10-09)


### Bug Fixes

* **desktop:** check names before creating new project ([ed30908](https://github.com/nednella/agentos/commit/ed30908bc9bd7298862ae2c35369b4b4bd9f5b47))
* **desktop:** keep config of set-up project with sections ([1ece786](https://github.com/nednella/agentos/commit/1ece7863c89bd58ee4e1a1a3bc600cc2d2703759))
* **desktop:** set up the entry of the current folder ([e321a5f](https://github.com/nednella/agentos/commit/e321a5f0644c8a849049145e52a7876f3302d879))
* **internal:** ask set-up sessions how branches take base changes ([f447e73](https://github.com/nednella/agentos/commit/f447e73c0c66522e84f008b36071b4dcaec0dacc))
* **internal:** delete branch with no worktree in default clean-up ([80d4321](https://github.com/nednella/agentos/commit/80d432104b41c4568e0df3460b0cfdc515d676fc))
* **internal:** fill empty queue_sections when setting up ([85a6a6a](https://github.com/nednella/agentos/commit/85a6a6a7673fccb5bd358617a6d72bcfea59fcbb))
* **internal:** leave out empty agent_command when saving ([c697a29](https://github.com/nednella/agentos/commit/c697a299c13db769f22d5176fabd19fd7cc92aa5))
* **internal:** tell set-up sessions to check for a GitHub origin ([085fd73](https://github.com/nednella/agentos/commit/085fd7379aaddef16ce52b06fedf76dd37b40b4c))
* **ui:** create GitHub repo only for project just made ([7fcb97e](https://github.com/nednella/agentos/commit/7fcb97ed7aa90c2c61399ecfbfdba199088e34b2))
* **ui:** start one set-up session per click ([47c83fe](https://github.com/nednella/agentos/commit/47c83feb07269bac23f18c9583a19350c687215d))

## [0.22.0](https://github.com/nednella/agentos/compare/v0.21.0...v0.22.0) (2026-10-09)


### Features

* **desktop:** record issues filed, clean-ups, evidence and waits ([802eeeb](https://github.com/nednella/agentos/commit/802eeeb7f84a6605af3d2532cd4add4cd00e0abd))
* **desktop:** record pull request events ([b4caab8](https://github.com/nednella/agentos/commit/b4caab8ab19741bd69cc09cd54807778bbf367b4))
* **stats:** append-only event log and in-memory rollup ([98f7f9d](https://github.com/nednella/agentos/commit/98f7f9de51b4a1076bec5e50532f5e60501807d9))
* **stats:** gzip closed days of the event log ([ec69fdd](https://github.com/nednella/agentos/commit/ec69fdd0b200af50b2b2a9b38bb8b9de91ccc047))
* **ui:** show pull requests, issues and session length in the ledger ([df25581](https://github.com/nednella/agentos/commit/df25581749fe7925b73f9110d4bda25d4676f693))


### Bug Fixes

* **build:** stop check script's Brave code sign clone from triggering App Management denials ([4de7859](https://github.com/nednella/agentos/commit/4de7859a919502f9eaad870f48973c1391a9e330))
* **desktop:** stop Brave code sign clone from triggering App Management denials ([60af8e7](https://github.com/nednella/agentos/commit/60af8e7af6ececc998f70cd47a0c85fe2ab0fc49))
* **ui:** drop back icon from news page header ([cc4723c](https://github.com/nednella/agentos/commit/cc4723c85b23fe573e588afbadc46100588d1ea2))

## [0.21.0](https://github.com/nednella/agentos/compare/v0.20.0...v0.21.0) (2026-10-09)


### Features

* **desktop:** fetch every tracked PR each minute ([2109022](https://github.com/nednella/agentos/commit/21090225ab6b8e652840d186929e849c48b88754))
* **desktop:** wake session when its PR conflicts with base ([6ffb9e2](https://github.com/nednella/agentos/commit/6ffb9e28844ef406ab8300ca2934bb8a443490a6))
* **internal:** add pr_conflict_command config key ([fbe7535](https://github.com/nednella/agentos/commit/fbe75353be5945d40f3351edf5c5e6fea2c4f5cb))

## [0.20.0](https://github.com/nednella/agentos/compare/v0.19.1...v0.20.0) (2026-10-09)


### Features

* **cli:** add news command ([aa71ad2](https://github.com/nednella/agentos/commit/aa71ad2519dc5f1798f642ee3efb5ad5860e3a74))
* **desktop:** fetch TLDR Dev newsletter in background ([97f7401](https://github.com/nednella/agentos/commit/97f7401df81778fe73283f009d40200538e7a032))
* **desktop:** offer retry when new project's GitHub repo fails ([ee5ce09](https://github.com/nednella/agentos/commit/ee5ce096633be22451c31640521860be749cf061))
* **desktop:** return full changelog ([02d03d5](https://github.com/nednella/agentos/commit/02d03d5b5beb0ec53a4ffdc434b029d115dd2ecd)), closes [#269](https://github.com/nednella/agentos/issues/269)
* **ui:** add news view with index and source pages ([f11658a](https://github.com/nednella/agentos/commit/f11658aa751d45f1b0f9394f29ee282b812b70a2))
* **ui:** reopen patch notes from palette and version ([2880664](https://github.com/nednella/agentos/commit/2880664526c6bf835552a4a0fcdf70a158262799)), closes [#269](https://github.com/nednella/agentos/issues/269)


### Bug Fixes

* **desktop:** say RemoveProject stops sessions ([2fdd8d9](https://github.com/nednella/agentos/commit/2fdd8d9e5e54707747138c6fdd72b0fbdad3195b))

## [0.19.1](https://github.com/nednella/agentos/compare/v0.19.0...v0.19.1) (2026-10-09)


### Bug Fixes

* **internal:** tell agents to raise browser window for owner ([1bedce7](https://github.com/nednella/agentos/commit/1bedce79b238fc7d5354681fa8b29b98096ec732))

## [0.19.0](https://github.com/nednella/agentos/compare/v0.18.0...v0.19.0) (2026-10-09)


### Features

* **desktop:** add Ledger to stats service ([33af077](https://github.com/nednella/agentos/commit/33af0776a9cf266be089d1745a9ab4be51279ffa))
* **desktop:** record prompts, sessions and agent time per project ([934e92a](https://github.com/nednella/agentos/commit/934e92a57057ecca4c6ed3272a924ec2b983d75e))
* **desktop:** return release notes since last run ([b986196](https://github.com/nednella/agentos/commit/b98619642b2ab10cfca72cf2fa022179f6718af5)), closes [#255](https://github.com/nednella/agentos/issues/255)
* **ui:** open stats view on stats tab, interruptions second ([31fd254](https://github.com/nednella/agentos/commit/31fd2542286055b6f97176070c47684ee4d3aedf))
* **ui:** show patch notes once after update ([091a8a9](https://github.com/nednella/agentos/commit/091a8a9c74305e34f74afc0e454fddc9f7789826)), closes [#255](https://github.com/nednella/agentos/issues/255)
* **ui:** show your work across projects in stats view ([4ad150f](https://github.com/nednella/agentos/commit/4ad150f733a570fcfab28dd36f01e7d2865929ae))


### Bug Fixes

* **desktop:** return empty patch notes as a list ([83e5454](https://github.com/nednella/agentos/commit/83e54544cb065aee582fcb82aa1856d8e98b7851)), closes [#255](https://github.com/nednella/agentos/issues/255)
* **ui:** shrink patch notes sheet ([5b768f7](https://github.com/nednella/agentos/commit/5b768f78212650941361e9c800c167cd7d438018)), closes [#255](https://github.com/nednella/agentos/issues/255)

## [0.18.0](https://github.com/nednella/agentos/compare/v0.17.1...v0.18.0) (2026-10-09)


### Features

* **desktop:** relabel issue dropped on another section ([f73aa4d](https://github.com/nednella/agentos/commit/f73aa4dc3e59c373b7eb4668d3ff27e7834897a2)), closes [#231](https://github.com/nednella/agentos/issues/231)
* **internal:** let queue sections limit drag and drop ([b0f387c](https://github.com/nednella/agentos/commit/b0f387c885203e3a5bc06c9335c5599f50160f44)), closes [#231](https://github.com/nednella/agentos/issues/231)
* **ui:** drag issues between queue sections ([6132b19](https://github.com/nednella/agentos/commit/6132b192f1c281f440edb744ec18b6dc0c15077f)), closes [#231](https://github.com/nednella/agentos/issues/231)

## [0.17.1](https://github.com/nednella/agentos/compare/v0.17.0...v0.17.1) (2026-10-09)


### Bug Fixes

* **desktop:** end browsers left by earlier run when app starts ([7ecfe27](https://github.com/nednella/agentos/commit/7ecfe27445c572cf58402feb0ea882dae0d7287d)), closes [#249](https://github.com/nednella/agentos/issues/249)
* **ui:** keep exited shell closed when panel reopens ([6b04c74](https://github.com/nednella/agentos/commit/6b04c74ccee4a9b9fdb72b6916117eeb00c2baae))

## [0.17.0](https://github.com/nednella/agentos/compare/v0.16.0...v0.17.0) (2026-10-08)


### Features

* **desktop:** brief setup session through its system prompt ([ee65935](https://github.com/nednella/agentos/commit/ee65935b7d8a3883073d0c9776ae49374dc7d3c6))
* **internal:** give every session one base prompt ([aec16a6](https://github.com/nednella/agentos/commit/aec16a6daf7e70b555480adc82a3d48025445589))
* **internal:** set new projects up with AGENTS.md ([f16786c](https://github.com/nednella/agentos/commit/f16786ce2215c1336365d5beebf70c733c59f020))
* **ui:** offer set up project for every project ([e0879b3](https://github.com/nednella/agentos/commit/e0879b30d320868c9ac5f60416d2adc34c1bbfdc))


### Bug Fixes

* **build:** stop check script's Brave from triggering App Management denials ([3d4a39c](https://github.com/nednella/agentos/commit/3d4a39cc24bf45a79540ea980e9f5452f5a66c30))
* **desktop:** stop Brave updater from triggering App Management denials ([e19849c](https://github.com/nednella/agentos/commit/e19849c8a37eca3c441fd74013bd44f5fa61cdbf))

## [0.16.0](https://github.com/nednella/agentos/compare/v0.15.0...v0.16.0) (2026-10-08)


### Features

* **desktop:** offer to set up project with no queue sections ([cbcc1ac](https://github.com/nednella/agentos/commit/cbcc1ac3012fa992c9c75909c9f72f86e27f63fc))
* **internal:** add basic agentos block for project setup ([6dceeae](https://github.com/nednella/agentos/commit/6dceeae6d5f7cdb456a511e88e62e50b4d468bf3))
* **ui:** explain setup offer in card ([52837b8](https://github.com/nednella/agentos/commit/52837b8358918fcff7c440367b82aee88eaec79c))
* **ui:** show setup offer in queue and palette ([f9eacb1](https://github.com/nednella/agentos/commit/f9eacb1a98dd188a6d24188c99a3105ffb09a599))


### Bug Fixes

* **internal:** limit setup help to repository side ([b6d45ec](https://github.com/nednella/agentos/commit/b6d45ec0f8b7bb28caac98689e06de0e9a64cc64))

## [0.15.0](https://github.com/nednella/agentos/compare/v0.14.1...v0.15.0) (2026-10-08)


### Features

* **cli:** add agentos setup help ([dbe09d9](https://github.com/nednella/agentos/commit/dbe09d91a2de66b13ca3242563c9d473ae41f22d))
* **desktop:** point every session to agentos setup help ([dac6f61](https://github.com/nednella/agentos/commit/dac6f612016a3346dc29a70ecdc2ad389ebceceb))
* **internal:** let queue section match every issue with "*" ([c1325b2](https://github.com/nednella/agentos/commit/c1325b2f40e74d7829db4acf3bc5a5d1c83a5ba0))
* **internal:** put issues no queue section matches in no section ([85b223d](https://github.com/nednella/agentos/commit/85b223d26c6d10afffc14860c82026bf42431b17))
* **internal:** reject queue section after "*" section ([503df36](https://github.com/nednella/agentos/commit/503df36c8787cd11c3964846d622873312a8b55a))
* **ui:** fold issues that match no queue section ([3cc1eb7](https://github.com/nednella/agentos/commit/3cc1eb7ce0b3b2d57556aad89e5d92ece3bfd35f))
* **ui:** zoom and pan images in evidence viewer ([2196142](https://github.com/nednella/agentos/commit/2196142376c2d2a334bb7b315caa826b65338d09))
* **ui:** zoom evidence image with left and right click ([0a1de46](https://github.com/nednella/agentos/commit/0a1de469faa43b63b0d85459dec4316c47e316ed))


### Bug Fixes

* **desktop:** keep whole issue title in session title ([dfcc5fb](https://github.com/nednella/agentos/commit/dfcc5fb3cecc04ed9c0e86774d9bca6a0ca56c84)), closes [#237](https://github.com/nednella/agentos/issues/237)

## [0.14.1](https://github.com/nednella/agentos/compare/v0.14.0...v0.14.1) (2026-10-08)


### Bug Fixes

* **desktop:** look up PR once per turn, not on every poll ([7210d90](https://github.com/nednella/agentos/commit/7210d9015be629d153124231f5d0befd109dac54))
* **desktop:** warn once when PR lookups fail alike ([4adbb0b](https://github.com/nednella/agentos/commit/4adbb0bbe1ecb20b23646953948baa2842247332))
* **ui:** show kill confirm below session stat bar ([6ac8d92](https://github.com/nednella/agentos/commit/6ac8d92d1b46c15804c933e63a9d6583785aafe3))
* **ui:** show session kill button in danger colour ([fba5281](https://github.com/nednella/agentos/commit/fba528148caa33b55c13faca402dbe6103e5126f))

## [0.14.0](https://github.com/nednella/agentos/compare/v0.13.0...v0.14.0) (2026-10-07)


### Features

* **desktop:** let agents close their browser window ([a796cf6](https://github.com/nednella/agentos/commit/a796cf62986a2367dd30cdda06cf243451b9f392)), closes [#218](https://github.com/nednella/agentos/issues/218)


### Bug Fixes

* **desktop:** forget browser whose connection dropped ([93d40d8](https://github.com/nednella/agentos/commit/93d40d8314b79f30929e807ba241c884b901e9a1)), closes [#218](https://github.com/nednella/agentos/issues/218)
* **desktop:** start browser without last run's pages ([cdf9b15](https://github.com/nednella/agentos/commit/cdf9b15f6c62ea0f279860ea5be52cba9f4d25fb)), closes [#218](https://github.com/nednella/agentos/issues/218)
* **desktop:** stop browser once its last window closes ([9e5abaf](https://github.com/nednella/agentos/commit/9e5abaf80daf7b6a65c1d6d2f3fbb8cc889dd4b6)), closes [#218](https://github.com/nednella/agentos/issues/218)

## [0.13.0](https://github.com/nednella/agentos/compare/v0.12.0...v0.13.0) (2026-10-07)


### Features

* **cli:** add --prompt to agentos new ([3c5ed3c](https://github.com/nednella/agentos/commit/3c5ed3c7d4a3d09052768e20e2d1511edf2bb064))
* **desktop:** tell claude sessions how to start a session ([49b77dc](https://github.com/nednella/agentos/commit/49b77dcb5702aa71ac5b691de4f4246e7d6b3cf5))

## [0.12.0](https://github.com/nednella/agentos/compare/v0.11.1...v0.12.0) (2026-10-07)


### Features

* **build:** draw terminal prompt app icon ([11701f5](https://github.com/nednella/agentos/commit/11701f58d9da4f51f85290113ace59d0abbc2e96))
* **cli:** let the agent raise its browser window with --front ([4d1bfe3](https://github.com/nednella/agentos/commit/4d1bfe300e14fe022a533aecc29cd2c1e1ddd1b5))
* **desktop:** dismiss a browser dialog left open for 5 seconds ([fc42dd9](https://github.com/nednella/agentos/commit/fc42dd9589bb7b639ae929f74414f79a055caf13))
* **desktop:** drive the session browser through chrome-devtools-mcp ([18d7779](https://github.com/nednella/agentos/commit/18d7779eb5463398ac2fad6c4f7e16e4097e4950))
* **desktop:** give each project's browser a fixed debugging port ([401a32b](https://github.com/nednella/agentos/commit/401a32b2d2942ce7a95cd826b989920e183fdde3))
* **desktop:** name the session in its browser window's title ([4eaa6de](https://github.com/nednella/agentos/commit/4eaa6de9f86006231df4f71aa1e8dd1a5a72b9ff))
* **desktop:** open a session's tabs in its own browser window ([858e506](https://github.com/nednella/agentos/commit/858e506984cd68a34f3ea260a4559e741d990d8e))
* **desktop:** open each session's page in its own headed browser window ([bf8da5c](https://github.com/nednella/agentos/commit/bf8da5cadbe7987bc5bba6287e91c6b49f5157b5))
* **desktop:** show a status summary in the headed browser tab ([f0e0fa2](https://github.com/nednella/agentos/commit/f0e0fa20843686236058e72b0a27f5319fe27a2d))


### Bug Fixes

* **desktop:** ask the browser to quit before killing it ([213b27b](https://github.com/nednella/agentos/commit/213b27bf70054c412adeae0bf5f6edb40380bb61))
* **desktop:** centre window buttons on top bar ([a3e873c](https://github.com/nednella/agentos/commit/a3e873c76116c973d2ef9aeef826f0150c28acc2))
* **ui:** pad top of queue list ([84d7abb](https://github.com/nednella/agentos/commit/84d7abb47bd23743a608d630ac449aee32cb194d))


### Performance Improvements

* **desktop:** start chrome-devtools-mcp straight from the npx cache ([582c990](https://github.com/nednella/agentos/commit/582c990f7ba3b109d00fecdf32ac6f2527828188))

## [0.11.1](https://github.com/nednella/agentos/compare/v0.11.0...v0.11.1) (2026-10-07)


### Bug Fixes

* **ui:** focus new shell once it exists ([3a59c16](https://github.com/nednella/agentos/commit/3a59c16736a6d932c95460d8bfd9206109c47c61))

## [0.11.0](https://github.com/nednella/agentos/compare/v0.10.2...v0.11.0) (2026-10-07)


### Features

* **desktop:** let settings change data folder ([c6da83b](https://github.com/nednella/agentos/commit/c6da83b81071c5bcc2e40d1146f14d26310f055a))
* **desktop:** start sessions with agent's default model ([51fdbb1](https://github.com/nednella/agentos/commit/51fdbb1f2197ad0ed083eaac26f25207b6fa7c14))
* **internal:** read model defaults from agent's own settings ([66f53eb](https://github.com/nednella/agentos/commit/66f53eba7f210bcde56bb591b40f21dfa74d51e3))
* **ui:** add data folder setting ([dad8cbb](https://github.com/nednella/agentos/commit/dad8cbbd9aa7e8fe2e7ca07f885b6771ce892c7a))
* **ui:** remove ⌘1–9 shortcuts for jumping to session ([ac2371a](https://github.com/nednella/agentos/commit/ac2371a24f06889d8a89e7afe1d500a8ad77b5d2))


### Bug Fixes

* **cli:** keep late hook from bringing dismissed session back ([a0a9447](https://github.com/nednella/agentos/commit/a0a94473c786c1592e156bd5e63a0935dff4caff))
* **desktop:** close session's browser tab when its row goes ([aa3935c](https://github.com/nednella/agentos/commit/aa3935c177b078175b89abefbe56f8b4b275dd02))
* **desktop:** drop pull request results for forgotten session ([aa179ec](https://github.com/nednella/agentos/commit/aa179ecc985b4cee140ed68e127455a4a0c1ae69))
* **desktop:** give each session its own temp folder ([76608ef](https://github.com/nednella/agentos/commit/76608ef5314338efcf75afd3132283d371ac0d68))
* **desktop:** remove expired row's evidence and pull request tracking ([294cecb](https://github.com/nednella/agentos/commit/294cecbbd7789f54ba61ac41c4b6b40cc002c70c))
* **internal:** give each session unique id ([fee1a86](https://github.com/nednella/agentos/commit/fee1a8611536949e0d729640902e7629eb5a94c2))
* **internal:** keep a session's identity out of new tmux servers ([a4247e1](https://github.com/nednella/agentos/commit/a4247e16ceb1071eec4f37811b89f8a47a6525a4))
* **internal:** let Claude sessions use app folders without asking ([8bbd2dd](https://github.com/nednella/agentos/commit/8bbd2dde7f2d1b4c4cb5497f81d1ad8f858249fe))
* **ui:** align status badge with stat bar ([a0e6865](https://github.com/nednella/agentos/commit/a0e686565d1d148f386de0b5e44bb07853ce8452))
* **ui:** move selection off killed session ([13dddc0](https://github.com/nednella/agentos/commit/13dddc02022329c84fdeb4530fb20243887478c6))
* **ui:** open shell list as dropdown over shell ([180bec1](https://github.com/nednella/agentos/commit/180bec102f63eb4c303998658b46053716a657f9))
* **ui:** place settings before search in top bar ([eab94c2](https://github.com/nednella/agentos/commit/eab94c2d48bb426cff91235b738c7fa0f35b6b1d))
* **ui:** remove keep-awake sign from top bar ([d6e3a8d](https://github.com/nednella/agentos/commit/d6e3a8d89d9f32a9e0837bec96f5f02d98009a78))

## [0.10.2](https://github.com/nednella/agentos/compare/v0.10.1...v0.10.2) (2026-10-07)


### Bug Fixes

* **ui:** even out gaps around session status row ([4a96f81](https://github.com/nednella/agentos/commit/4a96f8125feab4c6753a62968c7910184577adf2))
* **ui:** tint only title row of session header ([82c787b](https://github.com/nednella/agentos/commit/82c787baf026b6136349375890ebab1d6abe3243))

## [0.10.1](https://github.com/nednella/agentos/compare/v0.10.0...v0.10.1) (2026-10-07)


### Bug Fixes

* **ui:** hide kill button on ended session ([0c0344c](https://github.com/nednella/agentos/commit/0c0344cffe8d0ee2e24d2cebe053cdb171afeccf))
* **ui:** keep killed session as ended row ([c9490dc](https://github.com/nednella/agentos/commit/c9490dc5a07d2a382d74d69f4d58e8ded20ce8a2))

## [0.10.0](https://github.com/nednella/agentos/compare/v0.9.0...v0.10.0) (2026-10-06)


### Features

* **desktop:** look up the PR when an agent's turn ends ([fd6bffd](https://github.com/nednella/agentos/commit/fd6bffd93bcc21003307e368f35b6bcacbef1d64))
* **internal:** poll pull requests every 10s by default ([6013f46](https://github.com/nednella/agentos/commit/6013f46fc1415013d8ef8552e929ced8cb12eab3))
* **ui:** collapse the shell list into the shell header ([9d05432](https://github.com/nednella/agentos/commit/9d05432352a423345b6edb7990ff28d7d9f02148))
* **ui:** pad settings rows and sections ([3c4dd33](https://github.com/nednella/agentos/commit/3c4dd33f957033c7c1571ba019edba86625c12cd))
* **ui:** read the queue again on a project switch ([942bc1b](https://github.com/nednella/agentos/commit/942bc1b5dde7ef34f906b4e8eb0443046a42f91f))
* **ui:** read the queue again when it is shown or the window regains focus ([cfb44f5](https://github.com/nednella/agentos/commit/cfb44f5d5e97dd4a7c48ba0b5d01c3c384426163))
* **ui:** widen settings panel so every row shows ([d977955](https://github.com/nednella/agentos/commit/d977955fc0657ee4d970882f30aa6fa34bc0e598))


### Reverts

* widen settings panel so every row shows ([29a0500](https://github.com/nednella/agentos/commit/29a05008e9bca90fafc17991a02e3e7865cfcce1))

## [0.9.0](https://github.com/nednella/agentos/compare/v0.8.0...v0.9.0) (2026-10-06)


### Features

* **desktop:** open and close shells beyond the first ([796ee47](https://github.com/nednella/agentos/commit/796ee47358df3ae832e556456d927436706d1728))
* **internal:** name several shells per project ([927d463](https://github.com/nednella/agentos/commit/927d46386944f31c84e706a1576e86d27cb2df34))
* **ui:** drop live detail line from session rows ([75d1be9](https://github.com/nednella/agentos/commit/75d1be93a59ba08048fee7c79129f103e664c9a8))
* **ui:** drop live detail line from session top bar ([62198ac](https://github.com/nednella/agentos/commit/62198ac1c9592b505fa21b0d7133e2d1f1d5729e))
* **ui:** list a project's shells beside the shell terminal ([4a2e164](https://github.com/nednella/agentos/commit/4a2e16417e0983c058732b8cd5ade08355ae9023))
* **ui:** move PR controls under the timeline, show state word on rows ([93a27dd](https://github.com/nednella/agentos/commit/93a27ddf15c1d2a4a80f0a67864a90d5a1973db5))
* **ui:** pin a full-width New shell row and close shells with a bin ([d7517b3](https://github.com/nednella/agentos/commit/d7517b3e6e271e667c2c1a716b4256aeeb0c7dd6))
* **ui:** put state badge beside the title in the top bar ([7cd9953](https://github.com/nednella/agentos/commit/7cd9953abcf09cdbc382ebc0d88c11704a4dbb00))
* **ui:** put state badge left of the model in the top bar ([3e88768](https://github.com/nednella/agentos/commit/3e887681856c0bd19483546cee136af3db96f62a))


### Bug Fixes

* **desktop:** hide configured projects whose folder is missing ([06ce479](https://github.com/nednella/agentos/commit/06ce479fda7b06be494d86e7da4a24a9ee3674d0))
* **desktop:** store session evidence under data_dir ([abfbb78](https://github.com/nednella/agentos/commit/abfbb789e795e0a7a69d70977cb8a58dc4ec1429))

## [0.8.0](https://github.com/nednella/agentos/compare/v0.7.1...v0.8.0) (2026-10-06)


### Features

* **desktop:** pass {force} to the clean-up command ([f81633f](https://github.com/nednella/agentos/commit/f81633fc53549bb3ae929f190ab9c61cbc318c1b))
* **desktop:** save text scale, keep awake, browser and digest settings ([0be0a54](https://github.com/nednella/agentos/commit/0be0a54fd2c3648598a9d452ddc4d5748ff024b5))
* **internal:** add text scale config key and write browser and digest in place ([4ea9480](https://github.com/nednella/agentos/commit/4ea948009ffdfd8d9622ed7136bef6d0327ee384))
* **ui:** set text size, keep awake, browser and digest in settings ([83b7b3f](https://github.com/nednella/agentos/commit/83b7b3f915e661e3af636c0c2f21903e8d533498))
* **ui:** use a cog for settings and move it last in the top bar ([23ea56e](https://github.com/nednella/agentos/commit/23ea56ed70057b44cc3b7d8668e6241bfa386be8))


### Bug Fixes

* **desktop:** count a failed clean-up command as done when nothing is left ([e7da1a4](https://github.com/nednella/agentos/commit/e7da1a4f989e2809eca6dd9dcb6c57f67dd44502))
* **desktop:** read the current project from the config ([5f418c5](https://github.com/nednella/agentos/commit/5f418c5ff1fa02799c3d92c21b75b8cff84f2eb0))

## [0.7.1](https://github.com/nednella/agentos/compare/v0.7.0...v0.7.1) (2026-10-06)


### Bug Fixes

* **internal:** keep a linked file linked when saving it ([5bc658c](https://github.com/nednella/agentos/commit/5bc658cd3c84643067fc748f5fd1cb0700016c0c))

## [0.7.0](https://github.com/nednella/agentos/compare/v0.6.0...v0.7.0) (2026-10-06)


### Features

* **desktop:** clean up with the project's own command ([f291f9c](https://github.com/nednella/agentos/commit/f291f9c1f6f3c66d18846103218c6b7217dc836f))
* **desktop:** start issue sessions from a section's action ([85ee372](https://github.com/nednella/agentos/commit/85ee372dd634480ce68ae6390560458c5ca45484))
* **internal:** name every config key in full ([596fd90](https://github.com/nednella/agentos/commit/596fd90df3aa5c98e0f682f57af2997a74d06e12))
* **internal:** replace lanes with queue sections and actions ([7291e05](https://github.com/nednella/agentos/commit/7291e056ccac257d048563d1a2203dbeed52cb5e))
* **ui:** group the queue by section and offer each section's actions ([1cd543c](https://github.com/nednella/agentos/commit/1cd543c2d55a40d80df523885d4f9a3cf7709969))

## [0.6.0](https://github.com/nednella/agentos/compare/v0.5.0...v0.6.0) (2026-10-06)


### Features

* **cli:** remove harness command ([f677455](https://github.com/nednella/agentos/commit/f677455947d2cf777f0cfba42ff3b91160f7a6a9))
* **desktop:** add settings service ([7790516](https://github.com/nednella/agentos/commit/7790516792b8bf722075ae89f121414c3151eb78))
* **desktop:** remove harness check ([d7b9c03](https://github.com/nednella/agentos/commit/d7b9c0360e5fb839fc1925a984ccd720718adfb7))
* **desktop:** resume an ended session's conversation on a wake ([d04eb8b](https://github.com/nednella/agentos/commit/d04eb8b7f027709b2b784c99878743cd94d2a98d))
* **desktop:** run the digest on the first agent that is signed in ([454689b](https://github.com/nednella/agentos/commit/454689b96ebd2dd1fa9389c9b3f6060510c29a2f))
* **desktop:** save a project's clean-up settings ([b85a4f6](https://github.com/nednella/agentos/commit/b85a4f6d7e2859f11ad3ae8defe688fd8bdb9503))
* **desktop:** sessions report their branch, no built-in issue branch ([f36dbf2](https://github.com/nednella/agentos/commit/f36dbf247c98c861303aec93f3ee1d930e76f29f))
* **desktop:** set AGENTOS_ISSUE in issue sessions ([b4e24d4](https://github.com/nednella/agentos/commit/b4e24d4d28028dad924130ec116425b445a6561e))
* **desktop:** wake sessions with the project's commands ([776d620](https://github.com/nednella/agentos/commit/776d620f6545b9fb7f12c595bef596cb8303a52b))
* **internal:** add on_review and on_checks project keys ([e248fb9](https://github.com/nednella/agentos/commit/e248fb9288bb093c66937927b311ce5421d0d426))
* **internal:** add theme to the config file ([bd1a233](https://github.com/nednella/agentos/commit/bd1a2338d2d7443ba4d6a5ac38acde67d50bc130))
* **internal:** choose clean-up per event, per project ([cf3801c](https://github.com/nednella/agentos/commit/cf3801cb76c776fac2c84a90d4f44868b3c10d32))
* **internal:** record the agent's conversation id from hooks ([80a2719](https://github.com/nednella/agentos/commit/80a2719c546e11b357ee541437106a5baf506365))
* **ui:** add settings panel with theme choice ([0839d15](https://github.com/nednella/agentos/commit/0839d15ff138470146adc4b4eabdef2306298253))
* **ui:** add theme toggle button to top bar ([ded2396](https://github.com/nednella/agentos/commit/ded2396ff46765b88c49912ec9977a99fc864f4f))
* **ui:** remove harness check action and stats button ([c32fd5c](https://github.com/nednella/agentos/commit/c32fd5cb278628a748cd7ccd2946371fb7a034c6))
* **ui:** show a toast while a new project is created ([b6de651](https://github.com/nednella/agentos/commit/b6de65122032f65467ba6562b97e193c58595025))
* **ui:** show clean-up settings in the settings panel ([f32e749](https://github.com/nednella/agentos/commit/f32e749f9d4b02e04209bfc356cc9049796f5979))

## [0.5.0](https://github.com/nednella/agentos/compare/v0.4.0...v0.5.0) (2026-10-06)


### Features

* **ui:** follow system light and dark appearance ([c2efba9](https://github.com/nednella/agentos/commit/c2efba92d8ac435b0b9701871db55d315faef5fe))

## [0.4.0](https://github.com/nednella/agentos/compare/v0.3.1...v0.4.0) (2026-10-06)


### Features

* **desktop:** follow the branch and PR of sessions with no issue ([a061e63](https://github.com/nednella/agentos/commit/a061e632940213f3a78daa87e81be519ffa783b6))
* **internal:** record the agent's working directory from hooks ([ab8c4e6](https://github.com/nednella/agentos/commit/ab8c4e6d875255b71a23dc5cf858fb7a14645318))


### Bug Fixes

* **ui:** ignore detach shortcut when no session is in view ([8ccece1](https://github.com/nednella/agentos/commit/8ccece19f59c0e5e573433900b8d4a54b8c48838))
* **ui:** keep panel focus style off session clicks ([71e5c46](https://github.com/nednella/agentos/commit/71e5c46de9b649bbe1a84713062d16c9a083428b))

## [0.3.1](https://github.com/nednella/agentos/compare/v0.3.0...v0.3.1) (2026-10-06)


### Bug Fixes

* **desktop:** centre the queue and notes empty states ([a217711](https://github.com/nednella/agentos/commit/a2177115e76dac5cb0eb37fb7cf0aee4ed47e95f))

## [0.3.0](https://github.com/nednella/agentos/compare/v0.2.0...v0.3.0) (2026-10-06)


### Features

* **desktop:** create a new project from the app ([#106](https://github.com/nednella/agentos/issues/106)) ([3616e60](https://github.com/nednella/agentos/commit/3616e602d253c3e267cd7805e92da5582df69036))
* **desktop:** create a project with a folder, README and GitHub repo ([7fdb00e](https://github.com/nednella/agentos/commit/7fdb00eaf6dc406bab50e9fc210ef27fcfda34c8))
* **ui:** add New project to the project panel ([230fa38](https://github.com/nednella/agentos/commit/230fa38faf616f2554520e2474aa274f73ef4e24))


### Bug Fixes

* **desktop:** create the GitHub repo public ([b0aaee4](https://github.com/nednella/agentos/commit/b0aaee44364d11074aad6130f21beb8858295749))
* name the local app bundle agentos-dev.app; stop sessions killing by name ([#102](https://github.com/nednella/agentos/issues/102)) ([0ca3203](https://github.com/nednella/agentos/commit/0ca320396e8e1442fa073d7243a3568ba87c27b8))
* **ui:** send terminal resize once a drag settles ([8a853b3](https://github.com/nednella/agentos/commit/8a853b3ec85ae7060e0cc515129a2fd265cf181e))
* **ui:** send terminal resize once a drag settles ([#104](https://github.com/nednella/agentos/issues/104)) ([46c2efd](https://github.com/nednella/agentos/commit/46c2efd43644f465cb2f984e3c913297cd63b8cb))

## [0.2.0](https://github.com/nednella/agentos/compare/v0.1.0...v0.2.0) (2026-10-06)


### Features

* **agent:** launch claude code with agentos hooks ([b0e3be0](https://github.com/nednella/agentos/commit/b0e3be01d1da469ef502e96370752ab824486344))
* **agent:** pass model and effort when launching claude ([94601ba](https://github.com/nednella/agentos/commit/94601ba72221eb72b554a47fb6e2b359babafa85))
* **atomicfile:** write files by rename ([2553d8d](https://github.com/nednella/agentos/commit/2553d8d0b3a298991ab4ca33c61f371869da9e0e))
* build, release and self-update pipeline ([#76](https://github.com/nednella/agentos/issues/76)) ([9cdc583](https://github.com/nednella/agentos/commit/9cdc583a094d6114f197ea2db7f1c9c7a4461cd6))
* **bus:** receive hook events over unix socket ([b57c3dc](https://github.com/nednella/agentos/commit/b57c3dca91029c8100f330ac688851facd49e5dd))
* **cli:** add app commands ([3689a57](https://github.com/nednella/agentos/commit/3689a573ede92043ded3ab334b936200492b2c97))
* **cli:** add command skeleton ([d78c8f0](https://github.com/nednella/agentos/commit/d78c8f006b119ca12803aba9ff76fad9e1ebae83))
* **cli:** add digest commands ([3b6a63d](https://github.com/nednella/agentos/commit/3b6a63da938493e0195a2d6188b36d721c4c178e))
* **cli:** add note and stats commands ([8c0668e](https://github.com/nednella/agentos/commit/8c0668e17eec69750f77b6e0dc0b4c1294a89407))
* **cli:** add update command and daily release notice ([8164229](https://github.com/nednella/agentos/commit/8164229f0aa7db80d52a232727c303cbee83a206))
* **cli:** drive browser ([2758f16](https://github.com/nednella/agentos/commit/2758f16674e92cdb00357e7ab71ea9a99afb4841))
* **cli:** file evidence ([6792a6a](https://github.com/nednella/agentos/commit/6792a6a9f2dd6a0520bf5c2577636e351c5156d2))
* **cli:** make app bundle's binary the agentos command ([7df2f7d](https://github.com/nednella/agentos/commit/7df2f7df80e72ae045becf6580d75e77e1a90b5f))
* **cli:** report hook events ([efc4a18](https://github.com/nednella/agentos/commit/efc4a18c3963c148fb1c74f6983752369e5e8f5f))
* **cli:** stop sessions ([e320ee6](https://github.com/nednella/agentos/commit/e320ee626ea47ed112fc6adf0473565bd3362fb7))
* **control:** answer cli requests ([85416c8](https://github.com/nednella/agentos/commit/85416c8df7921272e99d2e7b1ad8babb584a32ac))
* **desktop:** add harness check ([35dcf3c](https://github.com/nednella/agentos/commit/35dcf3c2e3a3cd1e136ed76ab82200961edc30d7))
* **desktop:** add shell session ([31697d3](https://github.com/nednella/agentos/commit/31697d387757ea9b44a09aa8096bca937ade51d3))
* **desktop:** add streaming command runner ([6b46f4b](https://github.com/nednella/agentos/commit/6b46f4b1c994bef1f43ba11db836921c1ea1878a))
* **desktop:** add weekly digest ([248fa17](https://github.com/nednella/agentos/commit/248fa17eb3741f1e78d92b8598046c24fef131f4))
* **desktop:** add window skeleton ([2a9d4fa](https://github.com/nednella/agentos/commit/2a9d4fa5aee1cfd4ae3022006d3240f07f301ee2))
* **desktop:** attach note pictures when filing issue ([09f8a26](https://github.com/nednella/agentos/commit/09f8a267a91cc075795e5ebf94cc3e3c58922816))
* **desktop:** attach note pictures when filing issue ([#69](https://github.com/nednella/agentos/issues/69)) ([76c44b1](https://github.com/nednella/agentos/commit/76c44b16a69982060eb54fe7a5d3b1a7b5d9a37a))
* **desktop:** check releases and install a newer one ([8a4888a](https://github.com/nednella/agentos/commit/8a4888a57170a6f89bfa9593291e1a04aa5ceb68))
* **desktop:** collect evidence ([7c14059](https://github.com/nednella/agentos/commit/7c14059edd4a15555d9c1997e8024565a1fc9e8a))
* **desktop:** delete note once filed as issue ([9eae4cc](https://github.com/nednella/agentos/commit/9eae4cc3803cca7f040c1c231beb396175dcd862))
* **desktop:** drive browser over devtools protocol ([ec2bfd3](https://github.com/nednella/agentos/commit/ec2bfd3053397d94983dd51160c852ef733c29ac))
* **desktop:** end a project's sessions when forgetting it ([8215fce](https://github.com/nednella/agentos/commit/8215fceb64a74a296bcfdc77a3e65062f45582cf))
* **desktop:** end a project's sessions when forgetting it ([#93](https://github.com/nednella/agentos/issues/93)) ([560f527](https://github.com/nednella/agentos/commit/560f527c87926290c27b5c6596cdfcf74498bb33))
* **desktop:** expose browser commands to agents ([0f91b05](https://github.com/nednella/agentos/commit/0f91b0588dffb6a9785da39af7065f991ca4228d))
* **desktop:** keep the Mac awake while sessions work ([7bc3c17](https://github.com/nednella/agentos/commit/7bc3c17ece001b358e3c27a243ddc840a17e3fe6))
* **desktop:** list and start sessions ([e8bb0b6](https://github.com/nednella/agentos/commit/e8bb0b67cd8c82da24cbc2f5d91d81b7510a2c51))
* **desktop:** load issues from github ([b2f94a5](https://github.com/nednella/agentos/commit/b2f94a5b596d83133b533a145887b8df6ff5be6c))
* **desktop:** manage projects ([1b161d0](https://github.com/nednella/agentos/commit/1b161d05a8b6f205fba60d95a8f3799f73c25868))
* **desktop:** raise opened attention when session idles with new PR ([54e7084](https://github.com/nednella/agentos/commit/54e7084ac2ab2857f407ef213a61c11c74892a6e))
* **desktop:** read an issue's body and comments as GitHub renders them ([95fb683](https://github.com/nednella/agentos/commit/95fb683b8f137ddb7aaaa45d8a7586b5d310b1e5))
* **desktop:** record on clean-up entry whether PR was merged ([a65ac29](https://github.com/nednella/agentos/commit/a65ac29ac7394a77058bcbfaeab2385311616e64))
* **desktop:** send lane command when starting issue session ([bed3fa9](https://github.com/nednella/agentos/commit/bed3fa9d14e60d0325d773c22ddbd964256f3c0d)), closes [#21](https://github.com/nednella/agentos/issues/21)
* **desktop:** serve app commands over control socket ([7a69201](https://github.com/nednella/agentos/commit/7a6920138b7333e26d94486bd6f0c3151078335f))
* **desktop:** serve app to browser for testing ([bec2087](https://github.com/nednella/agentos/commit/bec2087c9751d0ed30a7aea6b2fd3751ec2c8453))
* **desktop:** store notes and pictures ([54fdb6c](https://github.com/nednella/agentos/commit/54fdb6c6f5307edf40a5a122a23200d2f1537635))
* **desktop:** stream session terminals ([f3d35b7](https://github.com/nednella/agentos/commit/f3d35b71cebb2d30ffcc260c1c89040463efadbe))
* **desktop:** tally interruptions ([2ee05f9](https://github.com/nednella/agentos/commit/2ee05f92c2dd77e357bf9fe72646258f755cd7da))
* **desktop:** track pull requests and clean up ([c1631db](https://github.com/nednella/agentos/commit/c1631dbbdcc10cdf6f8ee8a1bf63072e9d2e4242))
* **desktop:** watch pull requests and wake sessions on review feedback ([cadcd73](https://github.com/nednella/agentos/commit/cadcd736dc0302c5f023d87c05fa3c53a51a22a1))
* **internal:** add keep_awake config ([f3bd3f2](https://github.com/nednella/agentos/commit/f3bd3f2102f9c86655a13d768f964d9b0bf31af5))
* **internal:** add pr_watch and pr_poll project keys ([c87807f](https://github.com/nednella/agentos/commit/c87807f4a9c82deae54e159f53123fb3cd2dd15c))
* **internal:** add pull request wake prompts ([0a4cee0](https://github.com/nednella/agentos/commit/0a4cee0b1be338706c6eaf29ddcd524c16905c05))
* **internal:** add tmux Submit to press Enter in a session ([381a595](https://github.com/nednella/agentos/commit/381a5955b99f72b0e129f6e5d2e7da5c3082e94d))
* keep the Mac awake while sessions work ([#88](https://github.com/nednella/agentos/issues/88)) ([0313ac2](https://github.com/nednella/agentos/commit/0313ac215c4f94cc3d3bad734c4b7da565c3abb3))
* pick a model and effort per session ([#84](https://github.com/nednella/agentos/issues/84)) ([e26c7c8](https://github.com/nednella/agentos/commit/e26c7c83cddee35a659e6f7a8e38f3f2becd3da5))
* **project:** add model and effort config ([6cc5207](https://github.com/nednella/agentos/commit/6cc5207b4e18c86193e78410214b97f6289a6168))
* **project:** load and save project config ([3e26250](https://github.com/nednella/agentos/commit/3e2625034994f03fb8b6bc089fe72d2d42724b4f))
* **session:** model session state from hook events ([f1aca4d](https://github.com/nednella/agentos/commit/f1aca4db523586642835832c01f3656f6929b121))
* **tmux:** run sessions in private tmux server ([a5146d5](https://github.com/nednella/agentos/commit/a5146d590b03847152e9f4aa5400987b74e7782d))
* toast when session opens PR and goes idle ([#60](https://github.com/nednella/agentos/issues/60)) ([a33699f](https://github.com/nednella/agentos/commit/a33699f7acd31acd8d27b23a2638b3abf6e99e2b))
* **ui:** add action that detaches every session ([15e5ddb](https://github.com/nednella/agentos/commit/15e5ddb4edbe8b9ef5f1dc002cb6999e7e36c0d0)), closes [#72](https://github.com/nednella/agentos/issues/72)
* **ui:** add action that detaches selected session ([#75](https://github.com/nednella/agentos/issues/75)) ([7eb5e45](https://github.com/nednella/agentos/commit/7eb5e459abb4b1b8b0a07679492c9e14849d8373))
* **ui:** add app shell with top bar and panels ([8b2e697](https://github.com/nednella/agentos/commit/8b2e69730ba41411c8d60dc15ce23dc3b311e081))
* **ui:** add browser, evidence and digest views ([2e9b57c](https://github.com/nednella/agentos/commit/2e9b57c6aef911a7573dcda050dd9056ddc5e113))
* **ui:** add buttons that open project's GitHub repository ([47236f4](https://github.com/nednella/agentos/commit/47236f4d53f89d02c6b1f409041702e3757ac60e)), closes [#51](https://github.com/nednella/agentos/issues/51)
* **ui:** add notes ([d63a6d5](https://github.com/nednella/agentos/commit/d63a6d59beac890923aa08b903f6e1866b3f41d7))
* **ui:** add palette, shortcuts sheet and toasts ([39d0c55](https://github.com/nednella/agentos/commit/39d0c55e73a504ac8c76b676ef84cdb562af79dc))
* **ui:** add queue ([70c581f](https://github.com/nednella/agentos/commit/70c581f34c6b4d9ebc862a564d814d0cef9b307e))
* **ui:** add sessions panel and terminal ([d2c4677](https://github.com/nednella/agentos/commit/d2c4677e3ee1b938b112bb0c503fd9fb34ad10d9))
* **ui:** add shell strip ([6713076](https://github.com/nednella/agentos/commit/67130760c1dc3558245500c80e66a0eb35eb6d1a))
* **ui:** add stats view and lifecycle controls ([7130e9c](https://github.com/nednella/agentos/commit/7130e9ce68e481f597f98739ab111424401da257))
* **ui:** collapse queue head into search and filter rows ([b26d230](https://github.com/nednella/agentos/commit/b26d23013e31ef25725e2b2ddb07b6e2fdb04831))
* **ui:** collapse shell by clicking its header ([60999ed](https://github.com/nednella/agentos/commit/60999ed547d1b10f5b9d947a838c5a269e78d66d)), closes [#12](https://github.com/nednella/agentos/issues/12)
* **ui:** define backend contract and mock ([df3fc6e](https://github.com/nednella/agentos/commit/df3fc6eb87919a1f248617619ae5c550a5d38e94))
* **ui:** drop ⌘S shortcut for the shell ([3c28318](https://github.com/nednella/agentos/commit/3c28318de19659bff3a724f074330225ef5a233f)), closes [#13](https://github.com/nednella/agentos/issues/13)
* **ui:** drop note from list once filed as issue ([ff1356e](https://github.com/nednella/agentos/commit/ff1356e71ee546fbe93686b3307352efae8187cd))
* **ui:** frame note and session composers ([7e228ee](https://github.com/nednella/agentos/commit/7e228ee410ce35e73d4e423658deb6ebc810b00a)), closes [#20](https://github.com/nednella/agentos/issues/20)
* **ui:** give each key in a shortcut its own keycap ([471124a](https://github.com/nednella/agentos/commit/471124a5ab4ed5d174dc7931f46c65735d776455)), closes [#14](https://github.com/nednella/agentos/issues/14)
* **ui:** keep chosen done toast and drop candidates ([2f7a77c](https://github.com/nednella/agentos/commit/2f7a77c8864d1165f4339cb7648f65c32789a6c0))
* **ui:** open an issue in a larger view from the queue ([ab96f0d](https://github.com/nednella/agentos/commit/ab96f0de235b2680b198607c5a22c5cce3694150))
* **ui:** pad shell strip clear of window's rounded corners ([c28ea1b](https://github.com/nednella/agentos/commit/c28ea1bbf9e10da135e1e4897c1a769d43403e6e))
* **ui:** remove GitHub repo button from queue search ([4c40ce2](https://github.com/nednella/agentos/commit/4c40ce27dced27229f89b0d53beb4e67cea91c99)), closes [#56](https://github.com/nednella/agentos/issues/56)
* **ui:** remove GitHub repo button from queue search ([#57](https://github.com/nednella/agentos/issues/57)) ([f08eb38](https://github.com/nednella/agentos/commit/f08eb382d6b383e57979f21f2c3cb3adc4d9a640))
* **ui:** show a moon while the Mac is kept awake ([8f6bad7](https://github.com/nednella/agentos/commit/8f6bad7400626c3601398cf5bcb6977fb9655b23))
* **ui:** show an issue from the queue in a dialog ([e747202](https://github.com/nednella/agentos/commit/e747202e69708af344f5e5f06c3f528ac849c7c4)), closes [#31](https://github.com/nednella/agentos/issues/31)
* **ui:** show app name and tagline on empty session panel ([3984eca](https://github.com/nednella/agentos/commit/3984eca4551719e4b8b74e603b03a23d49229bf6)), closes [#77](https://github.com/nednella/agentos/issues/77)
* **ui:** show app name and tagline on empty session panel ([#82](https://github.com/nednella/agentos/issues/82)) ([ddd2d1d](https://github.com/nednella/agentos/commit/ddd2d1dd31088e1aaaee38681b9dccb89ba9c7d1))
* **ui:** show each session's model ([483ee98](https://github.com/nednella/agentos/commit/483ee98dac5dc305d12f8dd454d6dab43d290887))
* **ui:** show newer release in top bar with an update button ([d1bbbd1](https://github.com/nednella/agentos/commit/d1bbbd1ff00618c5527cd0c74efbb06f48b3a492))
* **ui:** show rewarding toast when merged work is cleaned up ([3384a70](https://github.com/nednella/agentos/commit/3384a70125bf3efacdac45144452b44480e51cd0))
* **ui:** show rewarding toast when merged work is cleaned up ([#73](https://github.com/nednella/agentos/issues/73)) ([6d3bde0](https://github.com/nednella/agentos/commit/6d3bde05b7aef937a3b272f376ed26d3dd981fd3))
* **ui:** show text size toast on zoom ([3bfac1e](https://github.com/nednella/agentos/commit/3bfac1ef743afb9044a2e7024ebde668bc3726b7)), closes [#8](https://github.com/nednella/agentos/issues/8)
* **ui:** show toast when session opens PR and goes idle ([f95d666](https://github.com/nednella/agentos/commit/f95d66676f9eb718373c80e54c617b71c777bed0))
* **ui:** space out collapsed sidebar strips ([51aee65](https://github.com/nednella/agentos/commit/51aee65ee54343cf41a035e4bf50c881ec0a19a8)), closes [#19](https://github.com/nednella/agentos/issues/19)
* **ui:** warn that forgetting a project ends its sessions ([2e6f99d](https://github.com/nednella/agentos/commit/2e6f99dd496dbab87106af4f9219897d4ed7e92e))


### Bug Fixes

* **browser:** report the scroll position once the page has moved ([ae425ac](https://github.com/nednella/agentos/commit/ae425ac85f645f10b643d61ad3b59c9f6f3cd2b2))
* **cli:** open app from bare command and lock hook state writes ([4822d44](https://github.com/nednella/agentos/commit/4822d4404bf191d1aaa0b309a238fd9aa4fde7ae))
* **desktop:** close review findings in services ([fca1151](https://github.com/nednella/agentos/commit/fca11513773ccf7dfab88bfbfbf72036b9f3d5f4))
* **desktop:** count only waiting sessions as needing you in project counts ([ef85088](https://github.com/nednella/agentos/commit/ef8508887e3a55b3992862631a22b706674947e8)), closes [#61](https://github.com/nednella/agentos/issues/61)
* **desktop:** count only waiting sessions as needing you in project counts ([#64](https://github.com/nednella/agentos/issues/64)) ([98e0f19](https://github.com/nednella/agentos/commit/98e0f19e25d6a87f879427292d7db4040304662a))
* **desktop:** keep known repo while issues refresh asks gh again ([afc95d6](https://github.com/nednella/agentos/commit/afc95d6270dcf69bb7ee9b2cc909983d3355ea7e)), closes [#52](https://github.com/nednella/agentos/issues/52)
* **desktop:** keep known repo while issues refresh asks gh again ([#55](https://github.com/nednella/agentos/issues/55)) ([72ff8cd](https://github.com/nednella/agentos/commit/72ff8cde8040928e68d56cada723319133ce9f66))
* **desktop:** open shell under one lock so concurrent calls share it ([8c5ca3c](https://github.com/nednella/agentos/commit/8c5ca3cf1287b0bbceed1ee07f62cacb51595c94))
* **desktop:** open shell under one lock so concurrent calls share it ([#81](https://github.com/nednella/agentos/issues/81)) ([43ebbad](https://github.com/nednella/agentos/commit/43ebbade3e20e5422cb4510373047846f5ed1995))
* **desktop:** reject with fixed text when repo has issues disabled ([a98bbdf](https://github.com/nednella/agentos/commit/a98bbdf5c595287111afeb1509e58e807dd2d855))
* **desktop:** remove the current project when the config lacks it ([cc9037f](https://github.com/nednella/agentos/commit/cc9037f65e9bcd8e4bd2e7207499c435df18ac52))
* **desktop:** remove the current project when the config lacks it ([#92](https://github.com/nednella/agentos/issues/92)) ([26b8f54](https://github.com/nednella/agentos/commit/26b8f54025e7c314f95edef10389145fcf54901a))
* **desktop:** trim error text in HTTP shim ([6a1c297](https://github.com/nednella/agentos/commit/6a1c297152ec20574e1949f14e1108d8d31bc1cb))
* **internal:** harden sessions, tmux targets, hooks and sockets ([3fdb507](https://github.com/nednella/agentos/commit/3fdb5072202af99aeee71eba6ecf3f23310c79a2))
* **ui:** centre empty state of sessions panel ([c55341a](https://github.com/nednella/agentos/commit/c55341acaebdf1d6c6eadc95f9cc710107b8841f))
* **ui:** centre issue dialog and narrow it ([6c2b8bf](https://github.com/nednella/agentos/commit/6c2b8bf209d10416ada6964565488ca34358b743)), closes [#65](https://github.com/nednella/agentos/issues/65)
* **ui:** centre issue dialog and narrow it ([#67](https://github.com/nednella/agentos/issues/67)) ([2463a4f](https://github.com/nednella/agentos/commit/2463a4f109c1027a3155ecaffcb929678d9fdaae))
* **ui:** centre issues-disabled notice and link the repo ([7517b9b](https://github.com/nednella/agentos/commit/7517b9be060ae896e4c08c503e3aeac1c9254338))
* **ui:** centre queue search icon and loosen its spacing ([9af1e4a](https://github.com/nednella/agentos/commit/9af1e4a0a43faae71be0b29599c3264cc926d3be)), closes [#49](https://github.com/nednella/agentos/issues/49)
* **ui:** centre toasts at the bottom of the window ([38e0cee](https://github.com/nednella/agentos/commit/38e0cee0775c82397388ff095fc80a799aa0ad6d))
* **ui:** close review findings ([fe37ca3](https://github.com/nednella/agentos/commit/fe37ca350a6ca196c98cedf32c18d07624ae62da))
* **ui:** colour working sessions green instead of blue ([2a8daae](https://github.com/nednella/agentos/commit/2a8daae829625be6a33656071fcf8d31f3cd291d))
* **ui:** colour working sessions green instead of blue ([#66](https://github.com/nednella/agentos/issues/66)) ([e9324f7](https://github.com/nednella/agentos/commit/e9324f7ff254c4d79f3bd0b8d3fdaaef49012a70))
* **ui:** detach only the selected session, on ⌘W ([33cd67b](https://github.com/nednella/agentos/commit/33cd67bc54d5c6236aaf82062df79094b1ba1256))
* **ui:** forget shell id when its session ends ([e9c10b0](https://github.com/nednella/agentos/commit/e9c10b006078503756dd831939a46bd1ccb257eb))
* **ui:** hide dead native scrollbar on xterm viewport ([abed344](https://github.com/nednella/agentos/commit/abed344d221c00872c5b4263a9de440b00c897df))
* **ui:** keep ⌘S; only its badge goes ([49a915b](https://github.com/nednella/agentos/commit/49a915b40fdef4977aa84a6004377456965ba7c6))
* **ui:** keep collapsed queue lanes collapsed across remounts ([eaad81d](https://github.com/nednella/agentos/commit/eaad81ddebf0906abe3ea636f9ce47645f2e6cfd))
* **ui:** keep kill button in place while confirming ([794ed37](https://github.com/nednella/agentos/commit/794ed370d0ce7e8295b76966e1e24ca9c9fb2d2c)), closes [#16](https://github.com/nednella/agentos/issues/16)
* **ui:** keep no xterm scrollback since tmux holds the history ([22e7e12](https://github.com/nednella/agentos/commit/22e7e122f069a404f56f72f49ff67dd129d2eb17)), closes [#59](https://github.com/nednella/agentos/issues/59)
* **ui:** keep note draft across tab switch and collapse ([4d1b215](https://github.com/nednella/agentos/commit/4d1b215edb2460557c750ff8c8b7f579bd7b444f))
* **ui:** keep terminal rows inside their padding ([8f9838e](https://github.com/nednella/agentos/commit/8f9838e4bcb31a67601161fe5e62a79da74b7f6d))
* **ui:** move note composer hint below the box ([11343a6](https://github.com/nednella/agentos/commit/11343a6b7e22957f783af1ea2241a2ab024f3e34)), closes [#42](https://github.com/nednella/agentos/issues/42)
* **ui:** open shell once from its placeholder button ([163ddc1](https://github.com/nednella/agentos/commit/163ddc1d669704d31abd8c46f401a1b0f113a77d))
* **ui:** remove xterm's dead scrollbar from terminals ([#68](https://github.com/nednella/agentos/issues/68)) ([32d3518](https://github.com/nednella/agentos/commit/32d3518ae88ba1479b493d76ee25c8ade12d664a))
* **ui:** resize shell from a thin divider, not its header ([d162808](https://github.com/nednella/agentos/commit/d162808e1620305379276fe734c164fe59ac07f0)), closes [#11](https://github.com/nednella/agentos/issues/11)
* **ui:** say issues are off instead of toasting gh's error ([f1f587b](https://github.com/nednella/agentos/commit/f1f587b9ae30dd397965671179ccececb20d0ab2)), closes [#78](https://github.com/nednella/agentos/issues/78)
* **ui:** say issues are off instead of toasting gh's error ([#80](https://github.com/nednella/agentos/issues/80)) ([24235ce](https://github.com/nednella/agentos/commit/24235ce64d00ce876846ec255f5f9c964b378ee9))
* **ui:** say pictures go with note when filing issue ([7dcf200](https://github.com/nednella/agentos/commit/7dcf20065cf02467ad028e27ce7493640c7c833e))
* **ui:** scroll the terminal one tmux line per cell of wheel travel ([4afbad9](https://github.com/nednella/agentos/commit/4afbad95c4a2501db14f4aaee864719a261c0a04)), closes [#15](https://github.com/nednella/agentos/issues/15)
* **ui:** show opened PR toast for selected session too ([ae65229](https://github.com/nednella/agentos/commit/ae652294b1e8a2e0e8dae072f2671b8e51bf0bfb))
* **ui:** style sessions empty state like a notice title ([b294163](https://github.com/nednella/agentos/commit/b294163e180dc97c642aadd6ab8ac18ea9283ea3))
* **ui:** use Ned's tagline on detached placeholder ([8134a9f](https://github.com/nednella/agentos/commit/8134a9fe42b0b8aa4cf8ca9f9d6c74a18d2046e2))


### Performance Improvements

* **ui:** refit terminal on every size change ([9192b0e](https://github.com/nednella/agentos/commit/9192b0ea35c980ac5a815b9efb4d695be8499aee)), closes [#10](https://github.com/nednella/agentos/issues/10)
