# hanfe

`hanfe`는 리눅스 evdev 키보드 이벤트를 가로채어 모든 환경(X11, Wayland, TTY)에서 사용할 수 있는 C++ 기반 한글 IME입니다. 실제 키보드 장치를 읽어 들여 두벌식 또는 세벌식(390) 자판 규칙에 맞춰 영어 입력을 한글 음절로 조합하고, 가상 uinput 키보드를 통해 다시 이벤트를 주입합니다. `Ctrl+Shift+U` 유니코드 시퀀스를 사용하므로 대부분의 애플리케이션에서 한글 입력이 동작하며, 필요하면 지정한 TTY에 직접 출력을 복제할 수도 있습니다.

## 주요 기능

- **두벌식 / 세벌식 390 자판 지원**: 두 레이아웃은 바이너리에 내장되어 있으며 실행 시 `--layout` 옵션으로 선택할 수 있습니다.
- **프리에딧 시뮬레이션**: 조합 중인 글자를 가상 키보드로 입력한 뒤 백스페이스로 교체하여 IME를 인식하지 않는 프로그램에도 한글을 전달합니다.
- **토글 키 지정**: `toggle.ini` 또는 `--toggle-config`로 한글/영문 전환 키를 설정할 수 있습니다. 기본값은 `KEY_RIGHTALT`와 `KEY_HANGUL` 조합입니다.
- **TTY 미러 출력**: `--tty /dev/ttyX` 옵션으로 조합된 결과를 특정 TTY에도 동시에 기록할 수 있습니다.

## 빌드

필요 패키지: `cmake`, `g++`(C++20 지원). Ubuntu 기준 설치 예시는 다음과 같습니다.

```bash
sudo apt-get update
sudo apt-get install build-essential cmake
```

그 다음 프로젝트를 빌드합니다.

```bash
cmake -S . -B build
cmake --build build
```

완료 후 실행 파일은 `build/hanfe`에 생성됩니다. 시스템 전역 설치가 필요하면 관리자 권한으로 다음을 실행할 수 있습니다.

```bash
sudo cmake --install build --prefix /usr
```

이는 `/usr/bin/hanfe`에 바이너리를 배치합니다.

## 실행 예시

```bash
sudo ./build/hanfe --device /dev/input/event3 --layout dubeolsik
```

주요 옵션:

- `--device PATH` : 가로챌 키보드 evdev 장치 경로 (필수)
- `--layout {dubeolsik|sebeolsik-390}` : 사용할 자판 (기본값 `dubeolsik`)
- `--toggle-config PATH` : 토글 키 설정 파일 지정 (기본값은 현재 디렉터리의 `toggle.ini`, 없으면 내장 기본값 사용)
- `--tty /dev/ttyX` : 결과를 지정한 TTY에도 동시에 출력
- `--list-layouts` : 지원 레이아웃 목록 출력 후 종료
- `-h, --help` : 사용법 출력

루트 권한이 있어야 대부분의 키보드 evdev 장치에 접근할 수 있습니다.

## 토글 키 설정

토글 파일은 간단한 INI 포맷을 사용합니다. 기본 제공되는 `toggle.ini` 예시는 다음과 같습니다.

```ini
[toggle]
keys = KEY_RIGHTALT, KEY_HANGUL
default_mode = hangul
```

- `keys` : 쉼표로 구분된 `KEY_*` 이름 목록입니다. 첫 번째로 눌린 토글 키가 입력 모드를 전환합니다.
- `default_mode` : 시작 모드 (`hangul` 또는 `latin`).

`alt_r`, `hangul`과 같은 일부 축약 이름도 사용할 수 있지만 가능하면 `KEY_*` 표기를 권장합니다.

## 레이아웃 메모

두벌식과 세벌식 390 매핑은 `src/layout.cpp`에 하드코딩되어 있습니다. 필요하다면 해당 파일을 수정하여 키 매핑을 조정하거나 새로운 레이아웃을 추가한 뒤 `load_layout` 함수를 확장할 수 있습니다.

## 주의 사항

- 장치를 독점(grab)하므로 실행 중에는 다른 IME가 동일 장치를 사용하지 않도록 하세요.
- `Ctrl+Shift+U` 기반 입력이 비활성화된 환경에서는 별도 설정이 필요할 수 있습니다.
- Wayland 보안 정책에 따라 루트 권한 또는 추가 권한 부여가 필요할 수 있습니다.
