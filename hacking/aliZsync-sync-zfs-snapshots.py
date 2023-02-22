#!/usr/bin/python3

import sys, argparse, os, time, datetime, subprocess, io, re, argparse

logLevel = 0

class ZFSFilesystem:
    def __init__(self, uri, sshIdentity = None):
        zfsRE = re.search (r'ssh://(?P<user>[^@]*)@(?P<server>[^:]*):((?P<port>[^:]*):)?(?P<sourcepath>.*)$', uri)

        if zfsRE:
            self.ssh = True
            self.sourcePath = zfsRE.group('sourcepath')
            self.user = zfsRE.group('user')
            self.server = zfsRE.group('server')
            self.port = zfsRE.group('port')
            self.sshIdentity = sshIdentity[0] if sshIdentity else None
        else:
            self.ssh = False
            self.sourcePath = uri

        logLevel > 0 and print (vars(self))


    def getZPoolFilesystem (self):
        return self.sourcePath


    def getZFSCmdLine (self, args):
        cmdLine = []

        if self.ssh:
            cmdLine.append ('ssh')

            if self.port:
                cmdLine.extend (['-p', self.port])

            if self.sshIdentity:
                cmdLine.extend (['-i', self.sshIdentity])

            # Disable StrictHostKeyChecking, as we run in a closed environment
            # TODO: make it an option
            cmdLine.extend (['-o', 'StrictHostKeyChecking=no'])

            cmdLine.append (self.user + '@' + self.server)

        cmdLine.extend (args)

        logLevel > 0 and print (["getZFSCmdLine:"] + cmdLine)

        return cmdLine


def main(argv):
    global logLevel

    parser = argparse.ArgumentParser(description='Synchronise all snapshots in a ZFS filesystem to another zfs filesystem.  Useful for synchronising backups.')

    parser.add_argument ("--debug", dest='debug', nargs='?', const=1, type=int, help='Debug level of the application.  Uses debug 1 if flag is passed without a number.')
    parser.add_argument ("--transport", dest='transport', nargs=1, help='transport to use for zfs send/receive, can be "mbuffer" (default) or "ssh"')
    parser.add_argument ("--sshIdentity", dest='sshIdentity', nargs=1, help='ssh identity key file to use when ssh-ing to destination servers')
    parser.add_argument ("source", help='Source ZFS filesystem.  Local filsystems are specified as zpool/filesystem.  Remote ones are specified as ssh://user@server[:port]:zpool/filesystem.')
    parser.add_argument ("destination", help='Destination ZFS filesystem.  Same format as source.')

    args = parser.parse_args()
    print ('args:' + str(args))

    if args.debug:
        logLevel = args.debug

    source = ZFSFilesystem(args.source, sshIdentity = args.sshIdentity)
    destination = ZFSFilesystem(args.destination, sshIdentity = args.sshIdentity)

    sourceZfsProc = subprocess.Popen(source.getZFSCmdLine (['/sbin/zfs', 'get', '-Hpd', '1', 'creation', source.getZPoolFilesystem ()]), stdout=subprocess.PIPE)
    sourceSubvolumesByCreation = readSubvolumesByCreation (sourceZfsProc)
    sourceCreationBySubvolumes = invertMap (sourceSubvolumesByCreation)
    sourceCreatedSorted = sorted (sourceSubvolumesByCreation)

    destinationZfsProc = subprocess.Popen(destination.getZFSCmdLine (['/sbin/zfs', 'get', '-Hpd', '1', 'creation', destination.getZPoolFilesystem ()]), stdout=subprocess.PIPE)
    destinationSubvolumesByCreation = readSubvolumesByCreation (destinationZfsProc)
    destinationCreationBySubvolumes = invertMap (destinationSubvolumesByCreation)

    for volIndex, volKey in enumerate(sourceCreatedSorted):
        sourceSubvolume = sourceSubvolumesByCreation[volKey]

        if sourceSubvolume in destinationCreationBySubvolumes:
            logLevel > 0 and print ("# Already present:", sourceSubvolume)
        else:
            if volIndex > 0:
                predecessorSubvolumeKey = sourceCreatedSorted[volIndex - 1]
                predecessorSubvolume = sourceSubvolumesByCreation[predecessorSubvolumeKey]

                print ("# Missing subvolume:", sourceSubvolume, " predecessor:", predecessorSubvolume)
                rawZfsSend = ['/sbin/zfs', 'send', '-i', source.getZPoolFilesystem () + '@' + predecessorSubvolume, source.getZPoolFilesystem () + '@' + sourceSubvolume]
                sendCmd = source.getZFSCmdLine (rawZfsSend)
                #print ("/sbin/zfs send -i "+ sourceZfsRoot + '@' + predecessorSubvolume + " " + sourceZfsRoot + '@' + sourceSubvolume + "| pv | ssh -p " + destinationPort + " " + " ".join(sshArgs) + " root@" + destinationServer + " /sbin/zfs receive -Fv " + destinationZfsRoot)
            else:
                print ("# Missing initial subvolume:", sourceSubvolume)
                rawZfsSend = ['/sbin/zfs', 'send', source.getZPoolFilesystem () + '@' + sourceSubvolume]
                sendCmd = source.getZFSCmdLine (rawZfsSend)
        #print ("/sbin/zfs send " + sourceZfsRoot + '@' + sourceSubvolume + "| pv | ssh -p " + destinationPort + " " + " ".join(sshArgs) + " root@" + destinationServer + " /sbin/zfs receive -Fv " + destinationZfsRoot)

            rawZfsReceive = ['/sbin/zfs', 'receive', '-Fv', destination.getZPoolFilesystem ()]
            receiveCmd = destination.getZFSCmdLine (['/sbin/zfs', 'receive', '-Fv', destination.getZPoolFilesystem ()])

            if args.transport == ['ssh']:
                fullCmdLine = ' '.join (sendCmd) + ' | dd | ' + ' '.join (receiveCmd)

                logLevel > 0 and print (fullCmdLine)
                result = subprocess.call(fullCmdLine, shell = True)

                if result:
                    print ("Error running:" + fullCmdLine)
                    sys.exit(1)
            else:
                fullReceiveLine = ' '.join (rawZfsReceive)
                fullSendLine = ' '.join (rawZfsSend)
                #first:
                # mbuffer -4 -s 128k -m 1G -I 9090 | zfs receive data/filesystem
                # then:
                # zfs send -i data/filesystem@1 data/filesystem@2 | mbuffer -s 128k -m 1G -O 10.0.0.1:9090
                remoteMvReceive = '"mbuffer -t -4 -s 128k -m 256M -I 47099 | ' + fullReceiveLine + '"'
                mbReceive = ' '.join (destination.getZFSCmdLine([remoteMvReceive]))
                print ("mbReceive:" + mbReceive)
                mbSend = fullSendLine + ' | mbuffer -t -s 128k -m 256M -O ' + destination.server + ':47099'
                print ("mbSend:" + mbSend)

                spReceive = subprocess.Popen(mbReceive, shell=True)
                print ("receive started")

                time.sleep(1) # we wait for the receiver to be ready

                spSend = subprocess.Popen(mbSend, shell=True)
                print ("send started")

                spSend.wait()
                spReceive.wait()

                fuserKill = ' '.join (destination.getZFSCmdLine(['"fuser -k -n tcp 47099"']))
                subprocess.call(fuserKill, shell=True)
                print ("all done")


    #print ("########## Local volumes #######")
    #printMap (sourceSubvolumesByCreation)


    #print ("########## Remote volumes #######")
    #printMap (destinationSubvolumesByCreation)




def readSubvolumesByCreation (zfsProc):
    subvolumesByCreation = { }

    for line in zfsProc.stdout.readlines ():
        # oneitbackups/current@0000-00-00_00:00 creation        1454195500      -
        theLine = line.decode ().strip ()
        lineRE = re.search( r'.*@(.*)\tcreation\t([0-9]*)\t.*$', theLine)
        #lineRE = re.search( r'.*@(.*)\tcreation\t', theLine)

        if lineRE:
            snapPath = lineRE.group(1)
            snapCreation = lineRE.group(2)

            subvolumesByCreation[snapCreation] = snapPath
        else:
            logLevel > 0 and print ("# Bad line:", theLine)

    return subvolumesByCreation



def invertMap (map):
    result = {}

    for key in sorted (map):
        result[map[key]] = key

    return result


def printMap (map):
    for key in sorted (map):
        print (key, map[key])



if __name__ == "__main__":
   main(sys.argv[1:])
   sys.exit(0)


