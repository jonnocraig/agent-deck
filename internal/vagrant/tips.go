package vagrant

import (
	"math/rand"
)

// Tip represents a helpful tip displayed to the user
type Tip struct {
	Text     string
	Source   string
	Category string // "vagrant" or "world"
}

// tips contains all 100 tips (50 vagrant, 50 world)
var tips = []Tip{
	// Vagrant Best Practices (50)
	{
		Text:     "Use NFS for synced folders on macOS/Linux for significant I/O speed improvements over default VirtualBox shared folders.",
		Source:   "vagrantup.com/docs/synced-folders/nfs",
		Category: "vagrant",
	},
	{
		Text:     "Use rsync synced folders as a high-performance, one-way sync alternative that works on all platforms.",
		Source:   "vagrantup.com/docs/synced-folders/rsync",
		Category: "vagrant",
	},
	{
		Text:     "Allocate adequate memory with `vb.memory` inside the provider block to prevent swapping and sluggish performance.",
		Source:   "vagrantup.com/docs/providers/virtualbox/configuration",
		Category: "vagrant",
	},
	{
		Text:     "Assign multiple CPU cores with `vb.cpus` to improve performance for multi-threaded workloads like compilation.",
		Source:   "vagrantup.com/docs/providers/virtualbox/configuration",
		Category: "vagrant",
	},
	{
		Text:     "Enable KVM paravirtualization for Linux guests with `--paravirtprovider kvm` for better timer accuracy.",
		Source:   "virtualbox.org/manual/ch03.html",
		Category: "vagrant",
	},
	{
		Text:     "Use linked clones (`vb.linked_clone = true`) in multi-machine setups to save gigabytes of disk space per VM.",
		Source:   "vagrantup.com/docs/providers/virtualbox/configuration",
		Category: "vagrant",
	},
	{
		Text:     "Always use `Vagrant.configure(\"2\")` to lock your Vagrantfile to the current configuration format.",
		Source:   "vagrantup.com/docs/vagrantfile",
		Category: "vagrant",
	},
	{
		Text:     "Relocate the `.vagrant` metadata directory by setting `VAGRANT_DOTFILE_PATH` to keep source trees clean.",
		Source:   "vagrantup.com/docs/other/environmental-variables",
		Category: "vagrant",
	},
	{
		Text:     "Use Ruby loops in the Vagrantfile to define multiple similar machines without duplicating config.",
		Source:   "vagrantup.com/docs/vagrantfile/tips",
		Category: "vagrant",
	},
	{
		Text:     "Automate pre/post actions with Vagrant triggers to streamline your workflow.",
		Source:   "vagrantup.com/docs/triggers",
		Category: "vagrant",
	},
	{
		Text:     "Pass sensitive data into your Vagrantfile via `ENV['VAR_NAME']` instead of hardcoding.",
		Source:   "vagrantup.com/docs/vagrantfile/tips",
		Category: "vagrant",
	},
	{
		Text:     "Pin your box version with `config.vm.box_version` to prevent breakage from upstream updates.",
		Source:   "vagrantup.com/docs/boxes/versioning",
		Category: "vagrant",
	},
	{
		Text:     "Set `run: \"once\"` on shell provisioners so setup scripts only execute on first `vagrant up`.",
		Source:   "vagrantup.com/docs/provisioning/shell",
		Category: "vagrant",
	},
	{
		Text:     "Use `inline` shell provisioners for short commands to keep everything in the Vagrantfile.",
		Source:   "vagrantup.com/docs/provisioning/shell",
		Category: "vagrant",
	},
	{
		Text:     "Run provisioning scripts as non-root with `privileged: false` to practice least-privilege.",
		Source:   "vagrantup.com/docs/provisioning/shell",
		Category: "vagrant",
	},
	{
		Text:     "Use Ansible as your provisioner for complex environments to leverage idempotent configuration.",
		Source:   "vagrantup.com/docs/provisioning/ansible",
		Category: "vagrant",
	},
	{
		Text:     "Force re-provisioning on a running machine with `vagrant provision` to apply updated scripts.",
		Source:   "vagrantup.com/docs/cli/provision",
		Category: "vagrant",
	},
	{
		Text:     "Run `vagrant rsync-auto` to automatically watch for host file changes and sync in near real-time.",
		Source:   "vagrantup.com/docs/cli/rsync-auto",
		Category: "vagrant",
	},
	{
		Text:     "Use a private network for stable host-only communication that survives VM reboots.",
		Source:   "vagrantup.com/docs/networking/private_network",
		Category: "vagrant",
	},
	{
		Text:     "Forward specific guest ports to the host with `forwarded_port` for accessing web services.",
		Source:   "vagrantup.com/docs/networking/forwarded_ports",
		Category: "vagrant",
	},
	{
		Text:     "Use a public (bridged) network to give the VM its own IP on your LAN.",
		Source:   "vagrantup.com/docs/networking/public_network",
		Category: "vagrant",
	},
	{
		Text:     "Set a hostname with `config.vm.hostname` so the VM is identifiable in shell prompts and DNS.",
		Source:   "vagrantup.com/docs/vagrantfile/machine_settings",
		Category: "vagrant",
	},
	{
		Text:     "Set `auto_correct: true` on forwarded ports so Vagrant resolves host port conflicts automatically.",
		Source:   "vagrantup.com/docs/networking/forwarded_ports",
		Category: "vagrant",
	},
	{
		Text:     "Enable verbose logging with `VAGRANT_LOG=info vagrant up` to diagnose startup and provisioning issues.",
		Source:   "vagrantup.com/docs/other/debugging",
		Category: "vagrant",
	},
	{
		Text:     "Debug SSH issues with `vagrant ssh -- -vvv` for verbose SSH client output.",
		Source:   "vagrantup.com/docs/cli/ssh",
		Category: "vagrant",
	},
	{
		Text:     "Enable VirtualBox GUI with `vb.gui = true` to see console output during kernel panics or boot errors.",
		Source:   "vagrantup.com/docs/providers/virtualbox/configuration",
		Category: "vagrant",
	},
	{
		Text:     "Use `vagrant reload` to restart the guest and apply Vagrantfile changes that require a reboot.",
		Source:   "vagrantup.com/docs/cli/reload",
		Category: "vagrant",
	},
	{
		Text:     "When all else fails, `vagrant destroy -f && vagrant up` rebuilds from scratch.",
		Source:   "vagrantup.com/docs/cli/destroy",
		Category: "vagrant",
	},
	{
		Text:     "Create reusable boxes from running VMs with `vagrant package --output my-custom.box`.",
		Source:   "vagrantup.com/docs/cli/package",
		Category: "vagrant",
	},
	{
		Text:     "Only use boxes from trusted sources like the official HashiCorp Cloud catalog.",
		Source:   "app.vagrantup.com/boxes/search",
		Category: "vagrant",
	},
	{
		Text:     "Resolve DNS issues inside the guest with `--natdnshostresolver1 on` in VirtualBox customizations.",
		Source:   "virtualbox.org/manual/ch08.html",
		Category: "vagrant",
	},
	{
		Text:     "On Windows, use the `vagrant-winnfsd` plugin for NFS-like synced folder performance.",
		Source:   "github.com/winnfsd/vagrant-winnfsd",
		Category: "vagrant",
	},
	{
		Text:     "Install `vagrant-vbguest` to keep Guest Additions in sync with your VirtualBox version automatically.",
		Source:   "github.com/dotless-de/vagrant-vbguest",
		Category: "vagrant",
	},
	{
		Text:     "Use `vagrant-cachier` to cache package downloads across `vagrant destroy` cycles.",
		Source:   "github.com/fgrehm/vagrant-cachier",
		Category: "vagrant",
	},
	{
		Text:     "Resize VM disks without manual VBoxManage commands using the `vagrant-disksize` plugin.",
		Source:   "github.com/sprotheroe/vagrant-disksize",
		Category: "vagrant",
	},
	{
		Text:     "Manage host `/etc/hosts` entries for VMs automatically with `vagrant-hostsupdater`.",
		Source:   "github.com/agiledivider/vagrant-hostsupdater",
		Category: "vagrant",
	},
	{
		Text:     "Audit installed plugins with `vagrant plugin list` and remove unused ones to stay lean.",
		Source:   "vagrantup.com/docs/cli/plugin",
		Category: "vagrant",
	},
	{
		Text:     "Define multiple machines in one Vagrantfile with `config.vm.define` blocks.",
		Source:   "vagrantup.com/docs/multi-machine",
		Category: "vagrant",
	},
	{
		Text:     "Target commands to specific machines like `vagrant ssh web` or `vagrant halt db`.",
		Source:   "vagrantup.com/docs/multi-machine",
		Category: "vagrant",
	},
	{
		Text:     "Designate a primary machine with `primary: true` so bare commands default to it.",
		Source:   "vagrantup.com/docs/multi-machine",
		Category: "vagrant",
	},
	{
		Text:     "Connect multi-machine setups via a private network on the same subnet.",
		Source:   "vagrantup.com/docs/multi-machine",
		Category: "vagrant",
	},
	{
		Text:     "Check for outdated boxes across all environments with `vagrant box outdated --global`.",
		Source:   "vagrantup.com/docs/cli/box",
		Category: "vagrant",
	},
	{
		Text:     "Update boxes with `vagrant box update`; the running VM uses the new box on next destroy+up.",
		Source:   "vagrantup.com/docs/cli/box",
		Category: "vagrant",
	},
	{
		Text:     "Improve network throughput with virtio-net adapters: `--nictype1 virtio`.",
		Source:   "vagrantup.com/docs/providers/virtualbox/configuration",
		Category: "vagrant",
	},
	{
		Text:     "Reclaim disk space by pruning old box versions with `vagrant box prune`.",
		Source:   "vagrantup.com/docs/cli/box",
		Category: "vagrant",
	},
	{
		Text:     "Create custom base boxes with pre-installed toolchains to standardize team onboarding.",
		Source:   "vagrantup.com/docs/boxes/base",
		Category: "vagrant",
	},
	{
		Text:     "Use `vagrant suspend/resume` for the fastest start/stop cycle -- state saved and restored in seconds.",
		Source:   "vagrantup.com/docs/cli/suspend",
		Category: "vagrant",
	},
	{
		Text:     "Prefer `vagrant halt` for a graceful shutdown when you want to free all host resources.",
		Source:   "vagrantup.com/docs/cli/halt",
		Category: "vagrant",
	},
	{
		Text:     "Take named snapshots with `vagrant snapshot save` before risky changes for instant rollback.",
		Source:   "vagrantup.com/docs/cli/snapshot",
		Category: "vagrant",
	},
	{
		Text:     "Run `vagrant global-status --prune` to find and clean orphaned VMs consuming resources.",
		Source:   "vagrantup.com/docs/cli/global-status",
		Category: "vagrant",
	},

	// World Facts (50)
	{
		Text:     "About 8% of human DNA is made of ancient viral sequences — remnants of retroviral infections embedded over millions of years.",
		Source:   "nature.com/scitable/topicpage/endogenous-retroviruses",
		Category: "world",
	},
	{
		Text:     "The placebo effect works even when people know they're taking a placebo; ritual and expectation alone produce measurable changes.",
		Source:   "nature.com/articles/s41598-017-19185-1",
		Category: "world",
	},
	{
		Text:     "Aerogel is over 90% air by volume, looks like 'frozen smoke,' and is used by NASA for extreme insulation.",
		Source:   "nasa.gov/mission_pages/stardust/multimedia/aerogel.html",
		Category: "world",
	},
	{
		Text:     "Ultra-thin gold films (nanometers thick) can transmit light, becoming transparent — a property bulk gold lacks.",
		Source:   "nature.com/articles/nnano.2016.256",
		Category: "world",
	},
	{
		Text:     "The 'Banana Equivalent Dose' is a real radiation unit — bananas contain enough potassium-40 to serve as a baseline.",
		Source:   "en.wikipedia.org/wiki/Banana_equivalent_dose",
		Category: "world",
	},
	{
		Text:     "The stoplight loosejaw dragonfish has rotatable red-light 'headlights' invisible to most deep-sea prey.",
		Source:   "mbari.org/animal/stoplight-loosejaw-dragonfish/",
		Category: "world",
	},
	{
		Text:     "Researchers documented octopuses punching fish hunting partners with no obvious benefit — possibly spite.",
		Source:   "cell.com/current-biology/fulltext/S0960-9822(22)00484-4",
		Category: "world",
	},
	{
		Text:     "Some sea slugs steal working chloroplasts from algae and keep them functioning — borrowing photosynthesis.",
		Source:   "nationalgeographic.com/animals/invertebrates/facts/sea-slugs",
		Category: "world",
	},
	{
		Text:     "The bone-eating worm Osedax has no mouth or stomach — it dissolves whale bones using symbiotic bacteria.",
		Source:   "mbari.org/animal/osedax/",
		Category: "world",
	},
	{
		Text:     "Certain caterpillar larvae stack prey corpses onto specialized back bristles as camouflage.",
		Source:   "nationalgeographic.com/animals/article/trash-carrying-larvae-insects",
		Category: "world",
	},
	{
		Text:     "The fungus Ophiocordyceps hijacks ant nervous systems, forcing them to climb and clamp at a precise height before erupting spores.",
		Source:   "nationalgeographic.com/animals/article/cordyceps-zombie-fungus-ants",
		Category: "world",
	},
	{
		Text:     "Leafcutter ants don't eat the leaves they harvest — they use them to cultivate underground fungus gardens.",
		Source:   "britannica.com/animal/leafcutter-ant",
		Category: "world",
	},
	{
		Text:     "Wombat poop is cube-shaped, likely preventing it from rolling away on slopes for more effective territory marking.",
		Source:   "nationalgeographic.com/animals/mammals/facts/wombat",
		Category: "world",
	},
	{
		Text:     "Some frogs survive being literally frozen solid by producing glucose cryoprotectants in their cells.",
		Source:   "britannica.com/animal/wood-frog",
		Category: "world",
	},
	{
		Text:     "Plants can boost chemical defenses when exposed to vibrations matching caterpillar chewing — a form of 'hearing.'",
		Source:   "scientificamerican.com/article/can-plants-hear/",
		Category: "world",
	},
	{
		Text:     "On exoplanet WASP-76b, iron vaporizes on the dayside and rains down as liquid iron on the cooler nightside.",
		Source:   "eso.org/public/news/eso1916/",
		Category: "world",
	},
	{
		Text:     "JWST identifies specific molecules in exoplanet atmospheres billions of miles away — turning starlight into chemistry.",
		Source:   "nasa.gov/mission/webb/",
		Category: "world",
	},
	{
		Text:     "Stars oscillate like instruments; asteroseismology studies these 'rings' to determine internal structure.",
		Source:   "esa.int/Science_Exploration/Space_Science/Asteroseismology",
		Category: "world",
	},
	{
		Text:     "A teaspoon of neutron star material would weigh roughly one billion tons on Earth.",
		Source:   "nasa.gov/universe/neutron-stars/",
		Category: "world",
	},
	{
		Text:     "The Moon drifts away from Earth at 3.8 cm/year — measured by bouncing lasers off Apollo reflectors.",
		Source:   "nasa.gov/moon-facts/",
		Category: "world",
	},
	{
		Text:     "A day on Venus (243 Earth days) is longer than its year (225 Earth days), and it rotates backwards.",
		Source:   "solarsystem.nasa.gov/planets/venus/",
		Category: "world",
	},
	{
		Text:     "The ISS orbits Earth every 90 minutes and is the third-brightest object in the night sky.",
		Source:   "spotthestation.nasa.gov/",
		Category: "world",
	},
	{
		Text:     "The Anglo-Zanzibar War of 1896 lasted 38-45 minutes — the shortest war in recorded history.",
		Source:   "britannica.com/event/Anglo-Zanzibar-War",
		Category: "world",
	},
	{
		Text:     "The first documented 'computer bug' was a literal moth found in a Harvard Mark II relay in 1947.",
		Source:   "computerhistory.org/tdih/september/9/",
		Category: "world",
	},
	{
		Text:     "Linear B was deciphered to reveal early Greek — overturning assumptions it was an unknown lost language.",
		Source:   "britannica.com/topic/Linear-B",
		Category: "world",
	},
	{
		Text:     "Peru's 'Boiling River' reaches scalding temperatures from deep geothermal heat, not volcanic lava.",
		Source:   "bbc.com/travel/article/20160518-the-amazon-rainforests-mysterious-boiling-river",
		Category: "world",
	},
	{
		Text:     "Death Valley's Racetrack Playa rocks slide on their own — explained by thin ice sheets pushed by wind.",
		Source:   "nps.gov/deva/learn/nature/racetrack.htm",
		Category: "world",
	},
	{
		Text:     "Earth has measurable 'gravity holes' — Canada's Hudson Bay has weaker gravity from glacial rebound effects.",
		Source:   "nasa.gov/mission_pages/GRACE/",
		Category: "world",
	},
	{
		Text:     "Lake Nyos in Cameroon 'burped' a lethal CO2 cloud in 1986, suffocating 1,700 people in minutes.",
		Source:   "britannica.com/event/Lake-Nyos-disaster",
		Category: "world",
	},
	{
		Text:     "Desert sand dunes can 'sing' — producing deep resonant hums audible for miles when sand avalanches.",
		Source:   "britannica.com/science/singing-sand",
		Category: "world",
	},
	{
		Text:     "New Zealand has the world's longest place name at 85 characters: Taumatawhakatangihanga...",
		Source:   "guinnessworldrecords.com/world-records/longest-place-name",
		Category: "world",
	},
	{
		Text:     "Earth's mantle holds water locked inside minerals — potentially more than all surface oceans combined.",
		Source:   "nature.com/articles/nature13080",
		Category: "world",
	},
	{
		Text:     "The 'liking gap': after conversations, people consistently underestimate how much strangers liked them.",
		Source:   "journals.sagepub.com/doi/10.1177/0956797618783714",
		Category: "world",
	},
	{
		Text:     "Most smartphone users experience 'phantom vibration syndrome' — feeling the phone buzz when it hasn't.",
		Source:   "psychologytoday.com/us/blog/brain-myths/201307/phantom-vibration-syndrome",
		Category: "world",
	},
	{
		Text:     "The 'decoy effect': adding an inferior third option measurably shifts preference toward a target choice.",
		Source:   "behavioraleconomics.com/resources/mini-encyclopedia-of-be/decoy-effect/",
		Category: "world",
	},
	{
		Text:     "The 'uncanny valley' makes near-perfect human replicas feel eerier than obviously fake ones.",
		Source:   "britannica.com/science/uncanny-valley",
		Category: "world",
	},
	{
		Text:     "The 'Mozart effect' is overstated — any measured boost is short-lived and linked to arousal, not intelligence.",
		Source:   "britannica.com/story/does-listening-to-mozart-really-make-you-smarter",
		Category: "world",
	},
	{
		Text:     "Candy banana flavor resembles the Gros Michel variety, nearly wiped out by fungus in the 1950s, not modern bananas.",
		Source:   "smithsonianmag.com/arts-culture/why-dont-banana-candies-taste-like-real-bananas",
		Category: "world",
	},
	{
		Text:     "Bacteriophages (viruses attacking bacteria) can alter cheese flavor by disrupting starter cultures during fermentation.",
		Source:   "asm.org/Articles/2019/December/Bacteriophages-and-the-Dairy-Industry",
		Category: "world",
	},
	{
		Text:     "Cold-brew coffee is chemically different from hot-brew — cold extraction shifts acidity and volatile compounds.",
		Source:   "acs.org/pressroom/presspacs/2018/acs-presspac-march-28-2018.html",
		Category: "world",
	},
	{
		Text:     "You can't 'taste' spicy — capsaicin activates heat/pain receptors, not taste buds.",
		Source:   "britannica.com/science/capsaicin",
		Category: "world",
	},
	{
		Text:     "The Netherlands has more bicycles than people and exports cycling infrastructure consulting worldwide.",
		Source:   "government.nl/topics/bicycles",
		Category: "world",
	},
	{
		Text:     "Blue Zone longevity research found social connection and purpose matter as much as diet for long life.",
		Source:   "nationalgeographic.com/magazine/article/secrets-of-long-life",
		Category: "world",
	},
	{
		Text:     "Some languages have no left/right — speakers use cardinal directions for everything and maintain remarkable orientation.",
		Source:   "pnas.org/doi/10.1073/pnas.0702920104",
		Category: "world",
	},
	{
		Text:     "Crows recognize individual human faces for years and socially communicate this to other crows — a collective 'grudge list.'",
		Source:   "scientificamerican.com/article/crows-never-forget-your-face/",
		Category: "world",
	},
	{
		Text:     "The vagus nerve is the literal wiring behind 'gut feelings' — a highway between gut and brain regulating mood.",
		Source:   "nature.com/articles/d41586-022-01043-4",
		Category: "world",
	},
	{
		Text:     "The longest tennis match lasted 11 hours 5 minutes across 3 days at Wimbledon 2010 (Isner vs Mahut, 70-68 final set).",
		Source:   "wimbledon.com/en_GB/news/articles/2019-06-24/the_longest_match.html",
		Category: "world",
	},
	{
		Text:     "The fastest red card in professional soccer was given within seconds of kickoff for violent conduct.",
		Source:   "guinnessworldrecords.com/world-records/fastest-red-card",
		Category: "world",
	},
	{
		Text:     "'Extreme ironing' is a real sport — competitors iron clothes on mountaintops, underwater, and while skydiving.",
		Source:   "bbc.com/news/uk-england-25823682",
		Category: "world",
	},
	{
		Text:     "In rare 'electrophonic' meteor events, observers hear crackling sounds simultaneously with seeing the meteor.",
		Source:   "britannica.com/science/meteor-astronomy",
		Category: "world",
	},
}

// GetRandomTip returns a random tip from the collection
func GetRandomTip() Tip {
	return tips[rand.Intn(len(tips))]
}

// GetNextTip returns the tip at the given index (wrapping around), for sequential rotation
func GetNextTip(index int) Tip {
	// Handle negative indices and wrap around
	normalizedIndex := index % len(tips)
	if normalizedIndex < 0 {
		normalizedIndex += len(tips)
	}
	return tips[normalizedIndex]
}
