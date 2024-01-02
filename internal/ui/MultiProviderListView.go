package ui

/*
================================================================================
Available Java Versions for Linux 64bit
================================================================================
 Vendor        | Use | Version      | Dist    | Status     | Identifier
--------------------------------------------------------------------------------
 Corretto      |     | 21.0.1       | amzn    |            | 21.0.1-amzn
               |     | 17.0.9       | amzn    |            | 17.0.9-amzn
               |     | 11.0.21      | amzn    |            | 11.0.21-amzn
               |     | 8.0.392      | amzn    |            | 8.0.392-amzn
 Dragonwell    |     | 17.0.9       | albba   |            | 17.0.9-albba
               |     | 11.0.20      | albba   |            | 11.0.20-albba
               |     | 8.0.382      | albba   |            | 8.0.382-albba
 Gluon         |     | 22.1.0.1.r17 | gln     |            | 22.1.0.1.r17-gln
               |     | 22.1.0.1.r11 | gln     |            | 22.1.0.1.r11-gln
 GraalVM CE    |     | 21.0.1       | graalce |            | 21.0.1-graalce
               |     | 17.0.9       | graalce |            | 17.0.9-graalce
 GraalVM Oracle|     | 21.0.1       | graal   |            | 21.0.1-graal
               |     | 17.0.9       | graal   |            | 17.0.9-graal
 Java.net      |     | 23.ea.3      | open    |            | 23.ea.3-open
               |     | 23.ea.2      | open    |            | 23.ea.2-open
               |     | 23.ea.1      | open    |            | 23.ea.1-open
               |     | 22.ea.29     | open    |            | 22.ea.29-open
               |     | 22.ea.28     | open    |            | 22.ea.28-open
               |     | 22.ea.27     | open    |            | 22.ea.27-open
               |     | 22.ea.26     | open    |            | 22.ea.26-open
               |     | 21.ea.35     | open    |            | 21.ea.35-open
 JetBrains     |     | 17.0.9       | jbr     |            | 17.0.9-jbr
               |     | 11.0.14.1    | jbr     |            | 11.0.14.1-jbr
 Liberica      |     | 21.0.1.crac  | librca  |            | 21.0.1.crac-librca
               |     | 21.0.1.fx    | librca  |            | 21.0.1.fx-librca
               |     | 21.0.1       | librca  |            | 21.0.1-librca
               |     | 17.0.9.crac  | librca  |            | 17.0.9.crac-librca
               |     | 17.0.9.fx    | librca  |            | 17.0.9.fx-librca
               |     | 17.0.9       | librca  |            | 17.0.9-librca
               |     | 11.0.21.fx   | librca  |            | 11.0.21.fx-librca
               |     | 11.0.21      | librca  |            | 11.0.21-librca
               |     | 8.0.392.fx   | librca  |            | 8.0.392.fx-librca
               |     | 8.0.392      | librca  |            | 8.0.392-librca
 Liberica NIK  |     | 23.1.1.r21   | nik     |            | 23.1.1.r21-nik
               |     | 22.3.4.r17   | nik     |            | 22.3.4.r17-nik
               |     | 22.3.4.r11   | nik     |            | 22.3.4.r11-nik
 Mandrel       |     | 23.r17       | mandrel | local only | 23.r17-mandrel
               |     | 23.1.1.r21   | mandrel |            | 23.1.1.r21-mandrel
 Microsoft     |     | 21.0.1       | ms      |            | 21.0.1-ms
               |     | 17.0.9       | ms      |            | 17.0.9-ms
               |     | 11.0.21      | ms      |            | 11.0.21-ms
 Oracle        |     | 21.0.1       | oracle  |            | 21.0.1-oracle
               |     | 17.0.9       | oracle  |            | 17.0.9-oracle
 SapMachine    |     | 21.0.1       | sapmchn |            | 21.0.1-sapmchn
               |     | 17.0.9       | sapmchn |            | 17.0.9-sapmchn
               |     | 11.0.21      | sapmchn |            | 11.0.21-sapmchn
 Semeru        |     | 17.0.9       | sem     |            | 17.0.9-sem
               |     | 11.0.21      | sem     |            | 11.0.21-sem
               |     | 8.0.392      | sem     |            | 8.0.392-sem
 Temurin       |     | 21.0.1       | tem     |            | 21.0.1-tem
               |     | 17.0.9       | tem     |            | 17.0.9-tem
               |     | 11.0.21      | tem     |            | 11.0.21-tem
               |     | 8.0.392      | tem     |            | 8.0.392-tem
 Tencent       |     | 17.0.9       | kona    |            | 17.0.9-kona
               |     | 11.0.21      | kona    |            | 11.0.21-kona
               |     | 8.0.392      | kona    |            | 8.0.392-kona
 Trava         |     | 11.0.15      | trava   |            | 11.0.15-trava
               |     | 8.0.282      | trava   |            | 8.0.282-trava
 Zulu          |     | 21.0.1       | zulu    |            | 21.0.1-zulu
               |     | 21.0.1.crac  | zulu    |            | 21.0.1.crac-zulu
               |     | 21.0.1.fx    | zulu    |            | 21.0.1.fx-zulu
               | >>> | 17.0.9       | zulu    | installed  | 17.0.9-zulu
               |     | 17.0.9.crac  | zulu    |            | 17.0.9.crac-zulu
               |     | 17.0.9.fx    | zulu    |            | 17.0.9.fx-zulu
               |     | 17.0.2       | zulu    | local only | 17.0.2-zulu
               |     | 17.0.0       | zulu    | local only | 17.0.0-zulu
               |     | 11.0.21      | zulu    |            | 11.0.21-zulu
               |     | 11.0.21.fx   | zulu    |            | 11.0.21.fx-zulu
               |     | 8.0.392      | zulu    |            | 8.0.392-zulu
               |     | 8.0.392.fx   | zulu    |            | 8.0.392.fx-zulu
               |     | 7.0.352      | zulu    |            | 7.0.352-zulu
               |     | 6.0.119      | zulu    |            | 6.0.119-zulu
 Unclassified  |     | 21.2.0.r16   | none    | local only | 21.2.0.r16-grl
================================================================================
Omit Identifier to install default version 21.0.1-tem:
    $ sdk install java
Use TAB completion to discover available versions
    $ sdk install java [TAB]
Or install a specific version by Identifier:
    $ sdk install java 21.0.1-tem
Hit Q to exit this list view
================================================================================
*/
