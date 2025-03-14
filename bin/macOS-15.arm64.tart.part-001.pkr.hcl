packer {
  required_plugins {
    lume = {
      version = ">= v0.0.1"
      # version = "= 0.0.1-dev"
      source = "github.com/warpbuilds/lume"
    }
  }
}

locals {
  image_folder = "/Users/${var.vm_username}/image-generation"
}

variable "vm_name" {
  type = string
}

variable "build_id" {
  type = string
}

variable "vm_username" {
  type      = string
  default   = "runner"
  sensitive = true
}

variable "vm_password" {
  type      = string
  default   = "runner"
  sensitive = true
}

variable "vcpu_count" {
  type    = number
  default = 6
}

variable "ram_size" {
  type    = number
  default = 8
}

variable "image_os" {
  type    = string
  default = "macos15"
}

variable "xcode_install_storage_url" {
  type    = string
  default = ""
}

variable "xcode_install_sas" {
  type    = string
  default = ""
}

variable "disk_size_gb" {
  type    = number
  default = 400
}

source "tart-cli" "tart" {
  vm_name      = "${var.vm_name}"
  cpu_count    = var.vcpu_count
  memory_gb    = var.ram_size
  disk_size_gb = var.disk_size_gb
  headless     = true
  ssh_password = var.vm_password
  ssh_username = var.vm_username
  ssh_timeout  = "120s"
}

build {
  sources = [
    "source.tart-cli.tart"
  ]

  provisioner "shell" {
    inline = [
      "sudo mkdir -p /Library/Application\\ Support/Tart",
      "sudo chown $(whoami) /Library/Application\\ Support/Tart"
    ]
  }

  provisioner "shell" {
    inline = ["mkdir ${local.image_folder}"]
  }

  provisioner "file" {
    destination = "${local.image_folder}/"
    sources = [
      "${path.root}/../scripts/tests",
      "${path.root}/../scripts/docs-gen",
      "${path.root}/../scripts/helpers"
    ]
  }

  provisioner "file" {
    destination = "${local.image_folder}/docs-gen/"
    source      = "${path.root}/../../../helpers/software-report-base"
  }

  provisioner "file" {
    destination = "${local.image_folder}/add-certificate.swift"
    source      = "${path.root}/../assets/add-certificate.swift"
  }

  provisioner "file" {
    destination = ".bashrc"
    source      = "${path.root}/../assets/bashrc"
  }

  provisioner "file" {
    destination = ".bash_profile"
    source      = "${path.root}/../assets/bashprofile"
  }

  provisioner "shell" {
    inline = ["mkdir ~/bootstrap"]
  }

  provisioner "file" {
    destination = "bootstrap"
    source      = "${path.root}/../assets/bootstrap-provisioner/"
  }

  provisioner "file" {
    destination = "${local.image_folder}/toolset.json"
    source      = "${path.root}/../toolsets/toolset-15.json"
  }

  provisioner "shell" {
    execute_command = "sudo sh -c '{{ .Vars }} {{ .Path }}'"
    inline = [
      "mv ${local.image_folder}/docs-gen ${local.image_folder}/software-report",
      "mkdir ~/utils",
      "mv ${local.image_folder}/helpers/invoke-tests.sh ~/utils",
      "mv ${local.image_folder}/helpers/utils.sh ~/utils"
    ]
  }

  provisioner "shell" {
    execute_command = "chmod +x {{ .Path }}; source $HOME/.bash_profile; {{ .Vars }} {{ .Path }}"
    scripts = [
      "${path.root}/../scripts/build/install-xcode-clt.sh",
      "${path.root}/../scripts/build/install-homebrew.sh",
      "${path.root}/../scripts/build/install-rosetta.sh"
    ]
  }

  provisioner "shell" {
    environment_vars = ["PASSWORD=${var.vm_password}", "USERNAME=${var.vm_username}"]
    execute_command  = "chmod +x {{ .Path }}; source $HOME/.bash_profile; sudo {{ .Vars }} {{ .Path }}"
    scripts = [
      "${path.root}/../scripts/build/configure-tccdb-macos.sh",
      "${path.root}/../scripts/build/configure-autologin.sh",
      "${path.root}/../scripts/build/configure-auto-updates.sh",
      "${path.root}/../scripts/build/configure-ntpconf.sh",
      "${path.root}/../scripts/build/configure-shell.sh"
    ]
  }

  provisioner "shell" {
    environment_vars = ["IMAGE_VERSION=${var.build_id}", "IMAGE_OS=${var.image_os}", "PASSWORD=${var.vm_password}"]
    execute_command  = "chmod +x {{ .Path }}; source $HOME/.bash_profile; {{ .Vars }} {{ .Path }}"
    scripts = [
      "${path.root}/../scripts/build/configure-preimagedata.sh",
      "${path.root}/../scripts/build/configure-ssh.sh",
      "${path.root}/../scripts/build/configure-machine.sh"
    ]
  }

  provisioner "shell" {
    execute_command   = "source $HOME/.bash_profile; sudo {{ .Vars }} {{ .Path }}"
    expect_disconnect = true
    inline            = ["echo 'Reboot VM'", "shutdown -r now"]
  }

  provisioner "shell" {
    environment_vars = ["USER_PASSWORD=${var.vm_password}", "IMAGE_FOLDER=${local.image_folder}"]
    execute_command  = "chmod +x {{ .Path }}; source $HOME/.bash_profile; {{ .Vars }} {{ .Path }}"
    pause_before     = "30s"
    scripts = [
      "${path.root}/../scripts/build/configure-windows.sh",
      "${path.root}/../scripts/build/install-powershell.sh",
      "${path.root}/../scripts/build/install-dotnet.sh",
      "${path.root}/../scripts/build/install-python.sh",
      "${path.root}/../scripts/build/install-azcopy.sh",
      "${path.root}/../scripts/build/install-openssl.sh",
      "${path.root}/../scripts/build/install-ruby.sh",
      "${path.root}/../scripts/build/install-rubygems.sh",
      "${path.root}/../scripts/build/install-git.sh",
      "${path.root}/../scripts/build/install-node.sh",
      "${path.root}/../scripts/build/install-common-utils.sh"
    ]
  }

  provisioner "shell" {
    environment_vars = ["XCODE_INSTALL_STORAGE_URL=${var.xcode_install_storage_url}", "XCODE_INSTALL_SAS=${var.xcode_install_sas}", "IMAGE_FOLDER=${local.image_folder}"]
    execute_command  = "chmod +x {{ .Path }}; source $HOME/.bash_profile; {{ .Vars }} pwsh -f {{ .Path }}"
    script           = "${path.root}/../scripts/build/Install-Xcode.ps1"
  }

  provisioner "shell" {
    execute_command   = "source $HOME/.bash_profile; sudo {{ .Vars }} {{ .Path }}"
    expect_disconnect = true
    inline            = ["echo 'Reboot VM'", "shutdown -r now"]
  }

  provisioner "shell" {
    environment_vars = ["IMAGE_FOLDER=${local.image_folder}"]
    execute_command  = "chmod +x {{ .Path }}; source $HOME/.bash_profile; {{ .Vars }} {{ .Path }}"
    scripts = [
      "${path.root}/../scripts/build/install-actions-cache.sh",
      "${path.root}/../scripts/build/install-llvm.sh",
      "${path.root}/../scripts/build/install-openjdk.sh",
      "${path.root}/../scripts/build/install-aws-tools.sh",
      "${path.root}/../scripts/build/install-rust.sh",
      "${path.root}/../scripts/build/install-gcc.sh",
      "${path.root}/../scripts/build/install-cocoapods.sh",
      "${path.root}/../scripts/build/install-android-sdk.sh",
      "${path.root}/../scripts/build/install-safari.sh",
      "${path.root}/../scripts/build/install-chrome.sh",
      "${path.root}/../scripts/build/install-bicep.sh",
      "${path.root}/../scripts/build/install-codeql-bundle.sh"
    ]
  }

  provisioner "shell" {
    environment_vars = ["IMAGE_FOLDER=${local.image_folder}"]
    execute_command  = "chmod +x {{ .Path }}; source $HOME/.bash_profile; {{ .Vars }} pwsh -f {{ .Path }}"
    scripts = [
      "${path.root}/../scripts/build/Install-Toolset.ps1",
      "${path.root}/../scripts/build/Configure-Toolset.ps1"
    ]
  }

  provisioner "shell" {
    environment_vars = ["IMAGE_FOLDER=${local.image_folder}"]
    execute_command  = "chmod +x {{ .Path }}; source $HOME/.bash_profile; {{ .Vars }} pwsh -f {{ .Path }}"
    script           = "${path.root}/../scripts/build/Configure-Xcode-Simulators.ps1"
  }

}