import Foundation
import Virtualization
let kernel = URL(fileURLWithPath: CommandLine.arguments[1])
let initrd = URL(fileURLWithPath: CommandLine.arguments[2])
let loader = VZLinuxBootLoader(kernelURL: kernel)
loader.initialRamdiskURL = initrd
loader.commandLine = "console=hvc0 rdinit=/init panic=-1 quiet"
let config = VZVirtualMachineConfiguration()
config.bootLoader = loader
config.cpuCount = 2
config.memorySize = 1024 * 1024 * 1024
let console = VZVirtioConsoleDeviceSerialPortConfiguration()
console.attachment = VZFileHandleSerialPortAttachment(fileHandleForReading: FileHandle.standardInput, fileHandleForWriting: FileHandle.standardOutput)
config.serialPorts = [console]
config.entropyDevices = [VZVirtioEntropyDeviceConfiguration()]
try config.validate()
let machine = VZVirtualMachine(configuration: config)
final class Delegate: NSObject, VZVirtualMachineDelegate {
 func guestDidStop(_ virtualMachine: VZVirtualMachine) { exit(0) }
 func virtualMachine(_ virtualMachine: VZVirtualMachine, didStopWithError error: Error) { fputs("VM error: \(error)\n", stderr); exit(1) }
}
let delegate = Delegate()
machine.delegate = delegate
machine.start { result in
 if case .failure(let error) = result { fputs("VM boot error: \(error)\n", stderr); exit(1) }
}
RunLoop.main.run()
