# Changelog

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
