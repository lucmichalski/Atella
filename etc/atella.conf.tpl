[agent]
  hostname = ""
  omit_hostname = false
  log_level = 2
  log_file = "/usr/local/atella/logs/atella.log"
  pid_file = "/usr/local/atella/atella.pid"
  host_cnt = 1
  hex_len = 10
  message_path = "/usr/local/atella/msg"
  master = false
  interval = 10
  net_timeout = 2
  
# [channels.TgSibnet]
#   address = "localhost"
#   port = 1
#   protocol = "tcp"
#   to = ["username"]
#   disabled = false
  
# [channels.Mail]
#   address = "localhost"
#   port = 25
#   auth = false
#   username = "user"
#   password = "password"
#   If ended with @hostname hostname will be replace to "hostname" parameter in 
#   agent section
#   from = "atella@hostname"
#   to = ["username@domain.com"]
#   disabled = false
