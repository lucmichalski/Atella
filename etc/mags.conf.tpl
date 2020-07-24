[agent]
  hostname = ""
  omit_hostname = false
  log_level = 4
  log_file = "/usr/local/mags/logs/mags.log"
  pid_file = "/var/run/mags.pid"
  host_cnt = 1
  hex_len = 10
  message_path = "/usr/local/mags/msg"
  
# [database]
#   type = "mysql"
#   host = "localhost"
#   port = 3306
#   dbname = "default"
#   user = "user"
#   password = "password"

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
#   from = "mags@hostname"
#   to = ["username@domain.com"]
#   disabled = false
  
# [sectors.sector1]
#   hosts = ["hostname"]
