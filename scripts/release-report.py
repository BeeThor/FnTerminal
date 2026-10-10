#!/usr/bin/env python3
"""Report only completed release gates; fnOS/device acceptance stays pending."""
import hashlib
import json
import platform
from pathlib import Path
import sys
import re
from datetime import datetime
from zoneinfo import ZoneInfo
version=json.loads(Path('package.json').read_text())['version']
browser=json.loads(Path('test-results/browser.json').read_text())
if browser.get('result')!='passed' or browser.get('version')!=version:
    raise SystemExit('Current-version browser evidence is missing')
for architecture in ['x86','arm']:
    artifact=Path(f'dist/FnTerminal-{version}-{architecture}.fpk')
    report=json.loads(artifact.with_suffix('.verification.json').read_text())
    if report.get('version')!=version or report.get('checks')!='passed' or report.get('sha256')!=hashlib.sha256(artifact.read_bytes()).hexdigest():
        raise SystemExit('FPK verification does not match current artifact')
vm=sys.argv[1]
if vm=='passed' and 'LINUX_TEST_RESULT=0' not in Path('test-results/linux.log').read_text():
    raise SystemExit('Linux VM runtime did not pass')
status={'version':version,'date':datetime.now(ZoneInfo('Asia/Shanghai')).strftime('%Y-%m-%d'),'local_release_gates':'passed check.sh and browser flow','check_host':platform.system()+'/'+platform.machine(),'browser_qa':'passed local real PTY fixture; simulated NAS identity','browser_checks':len(browser['checks']),'linux_runtime': 'passed isolated Alpine ARM64 VM' if vm=='passed' else 'not run by this release script','fpk_x86':'passed official fnpack and static archive verification','fpk_arm':'passed official fnpack and static archive verification','fnos_installation':'pending user Intel NAS regression','fnos_upgrade_uninstall_security':'pending','fnos_ssh':'pending actual NAS authentication','real_mobile_ime_clipboard':'pending','arm_fnos_installation':'pending','ci_remote_run':'pending'}
if browser.get('layout'): status['browser_layout_cases']=len(browser['layout']['metrics'])
status['fnos_desktop_ui_library']='unverified protected NAS desktop; FN Connect public UI is React/Semi'
status['ui_design']='official MIT Semi theme tokens mapped to Vue controls, not Semi React components'
check_log=Path('.cache/release/check.log').read_text()
counts=re.findall(r'ℹ tests (\d+)',check_log)
if counts: status['frontend_unit_tests']=int(counts[-1])
Path('dist/TEST_STATUS.json').write_text(json.dumps(status,ensure_ascii=False,indent=2)+'\n')
print(f'Release gates passed for {version}; actual fnOS acceptance remains pending')
