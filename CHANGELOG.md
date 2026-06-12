# Changelog

## [1.25.0](https://github.com/RomanAgaltsev/chaotic/compare/v1.24.0...v1.25.0) (2026-06-12)


### Features

* **adapter/io:** apply SlowReader/SlowWriter rate limiting ([8a076ed](https://github.com/RomanAgaltsev/chaotic/commit/8a076ed9d29b4582cb3b5fe8dfb3e009b9d61788))
* **adapter/io:** apply sticky Truncate cap ([f0a026c](https://github.com/RomanAgaltsev/chaotic/commit/f0a026cfb24de2a7ccd4dfb28e148a24f47e11f4))
* **adapter/io:** scaffold module with chaos_off passthrough ([9c7a33f](https://github.com/RomanAgaltsev/chaotic/commit/9c7a33f4878fc936a6fc298f4b52255b46a92628))
* **adapter/io:** wrap reader/writer with engine lifecycle ([6654e15](https://github.com/RomanAgaltsev/chaotic/commit/6654e157793f89a2df5deccc2f58a1e3a73c63c0))
* aws adapter ([#40](https://github.com/RomanAgaltsev/chaotic/issues/40)) ([d1d8873](https://github.com/RomanAgaltsev/chaotic/commit/d1d8873b700d31d7c5620d6cf1762246e1603977))
* bench testing ([#57](https://github.com/RomanAgaltsev/chaotic/issues/57)) ([831564b](https://github.com/RomanAgaltsev/chaotic/commit/831564ba510e0e94319d6ec44bfef414d971d279))
* built-in observers ([624b2f2](https://github.com/RomanAgaltsev/chaotic/commit/624b2f276143de355412e3e98479ad1a63f3dc5e))
* chaos package ([#17](https://github.com/RomanAgaltsev/chaotic/issues/17)) ([c0f4c41](https://github.com/RomanAgaltsev/chaotic/commit/c0f4c410d5f83560db08c91f004bd15bfdef1424))
* **engine:** add OpIO kind ([44f0477](https://github.com/RomanAgaltsev/chaotic/commit/44f0477b52700314056a9b7165a0561c359a4443))
* **engine:** serialize stream faults and warn on rate 0 ([8a142f2](https://github.com/RomanAgaltsev/chaotic/commit/8a142f20c55f91a86b0998735e7b3f6d78a3acd4))
* examples ([#19](https://github.com/RomanAgaltsev/chaotic/issues/19)) ([bbcf96c](https://github.com/RomanAgaltsev/chaotic/commit/bbcf96c25e7f17fd9069d3afccd013d678becd32))
* fault clock ([#61](https://github.com/RomanAgaltsev/chaotic/issues/61)) ([63a7edb](https://github.com/RomanAgaltsev/chaotic/commit/63a7edb54c7e7726a8d3a0a0f5030b893fc03473))
* **fault:** add SlowReader, SlowWriter, Truncate stream faults ([85532cb](https://github.com/RomanAgaltsev/chaotic/commit/85532cbbe95c02d6efd30993fd8c800236e7d337))
* hardening and ergonomics ([#15](https://github.com/RomanAgaltsev/chaotic/issues/15)) ([f27f06c](https://github.com/RomanAgaltsev/chaotic/commit/f27f06c9fd9482ee2ef144d85e8a86d6f9c3044f))
* http faults and golden chaos tests ([#23](https://github.com/RomanAgaltsev/chaotic/issues/23)) ([f7251d1](https://github.com/RomanAgaltsev/chaotic/commit/f7251d187cc62f9ff2bceee4b60a2d3ca67a57e3))
* kafka adapter ([#38](https://github.com/RomanAgaltsev/chaotic/issues/38)) ([2247631](https://github.com/RomanAgaltsev/chaotic/commit/22476315ed3dcfe78e599bc110414da9620ab00f))
* mongo adapter ([#36](https://github.com/RomanAgaltsev/chaotic/issues/36)) ([836f912](https://github.com/RomanAgaltsev/chaotic/commit/836f912aab068e3c743b9c20d505bb252cda3a9f))
* nats adapter ([#43](https://github.com/RomanAgaltsev/chaotic/issues/43)) ([6ece470](https://github.com/RomanAgaltsev/chaotic/commit/6ece4706547d537511fdb942d13d187eb81ef830))
* net adapter ([#46](https://github.com/RomanAgaltsev/chaotic/issues/46)) ([e3c6ae8](https://github.com/RomanAgaltsev/chaotic/commit/e3c6ae8992460515d08386f20c34d7dfa2fbbc5e))
* Outcome reporting and failure budget ([#2](https://github.com/RomanAgaltsev/chaotic/issues/2)) ([bde7155](https://github.com/RomanAgaltsev/chaotic/commit/bde7155bd5626013a76c6526e3a5c024a3172a87))
* pgx adapter implementation ([#25](https://github.com/RomanAgaltsev/chaotic/issues/25)) ([4d80c25](https://github.com/RomanAgaltsev/chaotic/commit/4d80c2587a9aeb675c4823fb27bacd4a3ff5d6ff))
* points analyzer ([#59](https://github.com/RomanAgaltsev/chaotic/issues/59)) ([2389608](https://github.com/RomanAgaltsev/chaotic/commit/2389608363835ded17102331e9c83e8c208c2f98))
* property testing ([#55](https://github.com/RomanAgaltsev/chaotic/issues/55)) ([872dc72](https://github.com/RomanAgaltsev/chaotic/commit/872dc72ec901cdc1234c888d5ad611d46e4f78a9))
* quick wins ([#65](https://github.com/RomanAgaltsev/chaotic/issues/65)) ([3d00197](https://github.com/RomanAgaltsev/chaotic/commit/3d0019711610347bb13f397e054868c3a9df6883))
* rabbitmq adapter ([#32](https://github.com/RomanAgaltsev/chaotic/issues/32)) ([aacf7e5](https://github.com/RomanAgaltsev/chaotic/commit/aacf7e50519a093e8c837a2f740888db8558da1c))
* redis adapter ([#30](https://github.com/RomanAgaltsev/chaotic/issues/30)) ([658d570](https://github.com/RomanAgaltsev/chaotic/commit/658d570b7974093ac76b7617696e31164e2b25a5))
* rule matchers and limits ([#49](https://github.com/RomanAgaltsev/chaotic/issues/49)) ([6295dc2](https://github.com/RomanAgaltsev/chaotic/commit/6295dc291971cd20ca865d93188b9da76676350d))
* rule sources ([#13](https://github.com/RomanAgaltsev/chaotic/issues/13)) ([e22e59b](https://github.com/RomanAgaltsev/chaotic/commit/e22e59b0d6fce1afc54117ec075b861ea83d5b14))
* rules dsl ([#51](https://github.com/RomanAgaltsev/chaotic/issues/51)) ([46eeaaf](https://github.com/RomanAgaltsev/chaotic/commit/46eeaaf6bd4262be67ebb501180d8e761c2bf4d9))
* safety rails ([#4](https://github.com/RomanAgaltsev/chaotic/issues/4)) ([55fb064](https://github.com/RomanAgaltsev/chaotic/commit/55fb0643844b517b1aade1bfba948739dc269aeb))
* scenarios ([#53](https://github.com/RomanAgaltsev/chaotic/issues/53)) ([a89971d](https://github.com/RomanAgaltsev/chaotic/commit/a89971d94fdd6cbe97a59211993171e9ebe1d08d))
* staged faults ([#63](https://github.com/RomanAgaltsev/chaotic/issues/63)) ([c73b728](https://github.com/RomanAgaltsev/chaotic/commit/c73b728487c39df623a0ffb0efc2e8ac003f5379))
* streaming faults ([69af70f](https://github.com/RomanAgaltsev/chaotic/commit/69af70f54033d895b695d17fb6be61aa0f8bb366))
* zero-cost chaos_off build tag ([#6](https://github.com/RomanAgaltsev/chaotic/issues/6)) ([69d369e](https://github.com/RomanAgaltsev/chaotic/commit/69d369ec536012243d570e61f0d3a7af695c48d2))


### Bug Fixes

* redis and rabbitmq review fixes ([#34](https://github.com/RomanAgaltsev/chaotic/issues/34)) ([4cb62ec](https://github.com/RomanAgaltsev/chaotic/commit/4cb62ec66ac5db85270438c413e3ccbc46458220))
* **release:** drop named component on root package ([#8](https://github.com/RomanAgaltsev/chaotic/issues/8)) ([bc39667](https://github.com/RomanAgaltsev/chaotic/commit/bc39667cbe52b248da2f1f4f1a05d80f26fe6287))
* review findings ([#27](https://github.com/RomanAgaltsev/chaotic/issues/27)) ([012f65c](https://github.com/RomanAgaltsev/chaotic/commit/012f65c3d942895ff8e256f5c84b15a1c06ab535))
* v3 review issues ([#21](https://github.com/RomanAgaltsev/chaotic/issues/21)) ([632872d](https://github.com/RomanAgaltsev/chaotic/commit/632872db994488fbd9f92f61c92ec3ad360f3e3b))

## [1.24.0](https://github.com/RomanAgaltsev/chaotic/compare/v1.23.0...v1.24.0) (2026-06-12)


### Features

* quick wins ([#65](https://github.com/RomanAgaltsev/chaotic/issues/65)) ([3d00197](https://github.com/RomanAgaltsev/chaotic/commit/3d0019711610347bb13f397e054868c3a9df6883))

## [1.23.0](https://github.com/RomanAgaltsev/chaotic/compare/v1.22.0...v1.23.0) (2026-06-12)


### Features

* staged faults ([#63](https://github.com/RomanAgaltsev/chaotic/issues/63)) ([c73b728](https://github.com/RomanAgaltsev/chaotic/commit/c73b728487c39df623a0ffb0efc2e8ac003f5379))

## [1.22.0](https://github.com/RomanAgaltsev/chaotic/compare/v1.21.0...v1.22.0) (2026-06-11)


### Features

* fault clock ([#61](https://github.com/RomanAgaltsev/chaotic/issues/61)) ([63a7edb](https://github.com/RomanAgaltsev/chaotic/commit/63a7edb54c7e7726a8d3a0a0f5030b893fc03473))

## [1.21.0](https://github.com/RomanAgaltsev/chaotic/compare/v1.20.0...v1.21.0) (2026-06-11)


### Features

* points analyzer ([#59](https://github.com/RomanAgaltsev/chaotic/issues/59)) ([2389608](https://github.com/RomanAgaltsev/chaotic/commit/2389608363835ded17102331e9c83e8c208c2f98))

## [1.20.0](https://github.com/RomanAgaltsev/chaotic/compare/v1.19.0...v1.20.0) (2026-06-11)


### Features

* bench testing ([#57](https://github.com/RomanAgaltsev/chaotic/issues/57)) ([831564b](https://github.com/RomanAgaltsev/chaotic/commit/831564ba510e0e94319d6ec44bfef414d971d279))

## [1.19.0](https://github.com/RomanAgaltsev/chaotic/compare/v1.18.0...v1.19.0) (2026-06-11)


### Features

* property testing ([#55](https://github.com/RomanAgaltsev/chaotic/issues/55)) ([872dc72](https://github.com/RomanAgaltsev/chaotic/commit/872dc72ec901cdc1234c888d5ad611d46e4f78a9))

## [1.18.0](https://github.com/RomanAgaltsev/chaotic/compare/v1.17.0...v1.18.0) (2026-06-11)


### Features

* scenarios ([#53](https://github.com/RomanAgaltsev/chaotic/issues/53)) ([a89971d](https://github.com/RomanAgaltsev/chaotic/commit/a89971d94fdd6cbe97a59211993171e9ebe1d08d))

## [1.17.0](https://github.com/RomanAgaltsev/chaotic/compare/v1.16.0...v1.17.0) (2026-06-11)


### Features

* rules dsl ([#51](https://github.com/RomanAgaltsev/chaotic/issues/51)) ([46eeaaf](https://github.com/RomanAgaltsev/chaotic/commit/46eeaaf6bd4262be67ebb501180d8e761c2bf4d9))

## [1.16.0](https://github.com/RomanAgaltsev/chaotic/compare/v1.15.0...v1.16.0) (2026-06-11)


### Features

* rule matchers and limits ([#49](https://github.com/RomanAgaltsev/chaotic/issues/49)) ([6295dc2](https://github.com/RomanAgaltsev/chaotic/commit/6295dc291971cd20ca865d93188b9da76676350d))

## [1.15.0](https://github.com/RomanAgaltsev/chaotic/compare/v1.14.0...v1.15.0) (2026-06-11)


### Features

* net adapter ([#46](https://github.com/RomanAgaltsev/chaotic/issues/46)) ([e3c6ae8](https://github.com/RomanAgaltsev/chaotic/commit/e3c6ae8992460515d08386f20c34d7dfa2fbbc5e))

## [1.14.0](https://github.com/RomanAgaltsev/chaotic/compare/v1.13.0...v1.14.0) (2026-06-11)


### Features

* nats adapter ([#43](https://github.com/RomanAgaltsev/chaotic/issues/43)) ([6ece470](https://github.com/RomanAgaltsev/chaotic/commit/6ece4706547d537511fdb942d13d187eb81ef830))

## [1.13.0](https://github.com/RomanAgaltsev/chaotic/compare/v1.12.0...v1.13.0) (2026-06-10)


### Features

* aws adapter ([#40](https://github.com/RomanAgaltsev/chaotic/issues/40)) ([d1d8873](https://github.com/RomanAgaltsev/chaotic/commit/d1d8873b700d31d7c5620d6cf1762246e1603977))

## [1.12.0](https://github.com/RomanAgaltsev/chaotic/compare/v1.11.0...v1.12.0) (2026-06-08)


### Features

* kafka adapter ([#38](https://github.com/RomanAgaltsev/chaotic/issues/38)) ([2247631](https://github.com/RomanAgaltsev/chaotic/commit/22476315ed3dcfe78e599bc110414da9620ab00f))

## [1.11.0](https://github.com/RomanAgaltsev/chaotic/compare/v1.10.1...v1.11.0) (2026-06-08)


### Features

* mongo adapter ([#36](https://github.com/RomanAgaltsev/chaotic/issues/36)) ([836f912](https://github.com/RomanAgaltsev/chaotic/commit/836f912aab068e3c743b9c20d505bb252cda3a9f))

## [1.10.1](https://github.com/RomanAgaltsev/chaotic/compare/v1.10.0...v1.10.1) (2026-06-06)


### Bug Fixes

* redis and rabbitmq review fixes ([#34](https://github.com/RomanAgaltsev/chaotic/issues/34)) ([4cb62ec](https://github.com/RomanAgaltsev/chaotic/commit/4cb62ec66ac5db85270438c413e3ccbc46458220))

## [1.10.0](https://github.com/RomanAgaltsev/chaotic/compare/v1.9.0...v1.10.0) (2026-06-05)


### Features

* rabbitmq adapter ([#32](https://github.com/RomanAgaltsev/chaotic/issues/32)) ([aacf7e5](https://github.com/RomanAgaltsev/chaotic/commit/aacf7e50519a093e8c837a2f740888db8558da1c))

## [1.9.0](https://github.com/RomanAgaltsev/chaotic/compare/v1.8.1...v1.9.0) (2026-06-05)


### Features

* redis adapter ([#30](https://github.com/RomanAgaltsev/chaotic/issues/30)) ([658d570](https://github.com/RomanAgaltsev/chaotic/commit/658d570b7974093ac76b7617696e31164e2b25a5))

## [1.8.1](https://github.com/RomanAgaltsev/chaotic/compare/v1.8.0...v1.8.1) (2026-06-04)


### Bug Fixes

* review findings ([#27](https://github.com/RomanAgaltsev/chaotic/issues/27)) ([012f65c](https://github.com/RomanAgaltsev/chaotic/commit/012f65c3d942895ff8e256f5c84b15a1c06ab535))

## [1.8.0](https://github.com/RomanAgaltsev/chaotic/compare/v1.7.0...v1.8.0) (2026-06-04)


### Features

* pgx adapter implementation ([#25](https://github.com/RomanAgaltsev/chaotic/issues/25)) ([4d80c25](https://github.com/RomanAgaltsev/chaotic/commit/4d80c2587a9aeb675c4823fb27bacd4a3ff5d6ff))

## [1.7.0](https://github.com/RomanAgaltsev/chaotic/compare/v1.6.1...v1.7.0) (2026-06-04)


### Features

* http faults and golden chaos tests ([#23](https://github.com/RomanAgaltsev/chaotic/issues/23)) ([f7251d1](https://github.com/RomanAgaltsev/chaotic/commit/f7251d187cc62f9ff2bceee4b60a2d3ca67a57e3))

## [1.6.1](https://github.com/RomanAgaltsev/chaotic/compare/v1.6.0...v1.6.1) (2026-06-04)


### Bug Fixes

* v3 review issues ([#21](https://github.com/RomanAgaltsev/chaotic/issues/21)) ([632872d](https://github.com/RomanAgaltsev/chaotic/commit/632872db994488fbd9f92f61c92ec3ad360f3e3b))

## [1.6.0](https://github.com/RomanAgaltsev/chaotic/compare/v1.5.0...v1.6.0) (2026-06-03)


### Features

* examples ([#19](https://github.com/RomanAgaltsev/chaotic/issues/19)) ([bbcf96c](https://github.com/RomanAgaltsev/chaotic/commit/bbcf96c25e7f17fd9069d3afccd013d678becd32))

## [1.5.0](https://github.com/RomanAgaltsev/chaotic/compare/v1.4.0...v1.5.0) (2026-06-03)


### Features

* chaos package ([#17](https://github.com/RomanAgaltsev/chaotic/issues/17)) ([c0f4c41](https://github.com/RomanAgaltsev/chaotic/commit/c0f4c410d5f83560db08c91f004bd15bfdef1424))

## [1.4.0](https://github.com/RomanAgaltsev/chaotic/compare/v1.3.0...v1.4.0) (2026-06-03)


### Features

* hardening and ergonomics ([#15](https://github.com/RomanAgaltsev/chaotic/issues/15)) ([f27f06c](https://github.com/RomanAgaltsev/chaotic/commit/f27f06c9fd9482ee2ef144d85e8a86d6f9c3044f))

## [1.3.0](https://github.com/RomanAgaltsev/chaotic/compare/v1.2.0...v1.3.0) (2026-06-02)


### Features

* rule sources ([#13](https://github.com/RomanAgaltsev/chaotic/issues/13)) ([e22e59b](https://github.com/RomanAgaltsev/chaotic/commit/e22e59b0d6fce1afc54117ec075b861ea83d5b14))

## [1.2.0](https://github.com/RomanAgaltsev/chaotic/compare/v1.1.0...v1.2.0) (2026-06-01)


### Features

* built-in observers ([624b2f2](https://github.com/RomanAgaltsev/chaotic/commit/624b2f276143de355412e3e98479ad1a63f3dc5e))
* zero-cost chaos_off build tag ([#6](https://github.com/RomanAgaltsev/chaotic/issues/6)) ([69d369e](https://github.com/RomanAgaltsev/chaotic/commit/69d369ec536012243d570e61f0d3a7af695c48d2))


### Bug Fixes

* **release:** drop named component on root package ([#8](https://github.com/RomanAgaltsev/chaotic/issues/8)) ([bc39667](https://github.com/RomanAgaltsev/chaotic/commit/bc39667cbe52b248da2f1f4f1a05d80f26fe6287))

## [1.1.0](https://github.com/RomanAgaltsev/chaotic/compare/v1.0.0...v1.1.0) (2026-05-31)


### Features

* safety rails ([#4](https://github.com/RomanAgaltsev/chaotic/issues/4)) ([55fb064](https://github.com/RomanAgaltsev/chaotic/commit/55fb0643844b517b1aade1bfba948739dc269aeb))

## 1.0.0 (2026-05-30)


### Features

* Outcome reporting and failure budget ([#2](https://github.com/RomanAgaltsev/chaotic/issues/2)) ([bde7155](https://github.com/RomanAgaltsev/chaotic/commit/bde7155bd5626013a76c6526e3a5c024a3172a87))

## Changelog

All notable changes to this project are documented here. This file is
maintained by [release-please](https://github.com/googleapis/release-please)
from Conventional Commit messages.
