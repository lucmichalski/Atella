class Host < ApplicationRecord

  def wrap_version(code)
    res = false
    self.version = "unknown"
    unless code.nil?
      begin
        s = TCPSocket.open(self.address, 5223)
        s.puts(code)
        loop do
          line = s.gets
          break if (line == "Bye!" || line.nil?)
          l = line.rstrip.split
          case (l[0])
            when "Bye!"
              break
            when "canTalk"
              s.puts("version")
              self.version = s.gets.split[1]
              unless self.version.nil?
                self.version.rstrip!  
                res = true
              end
              s.puts("exit")
              break
            else
              s.puts("exit")
              break
          end
        end
        s.close
      rescue => v
        STDERR.print("#{v}\n")
        res = false
      end
    end
    return res
  end

  def Host.reload_hosts_config
    settings = Rails.application.config.atella
    error = nil
    begin
      mastersConfig = TOML.load_file(settings["atella"]["masterServersConfig"])
      sectorsConfig = TOML.load_file(settings["atella"]["sectorsConfig"])
    rescue => v
      error = v
    end
    
    if error 
      STDERR.print("Fatal: #{error}\n")  
    end
    unless sectorsConfig["sectors"].nil?
      sectorsConfig["sectors"].each do |sec|
        unless sec.length < 2
          unless sec[1]["hosts"].nil?
            sec[1]["hosts"].each do |host|
              host = host.split()
              unless host.length < 2
                version = "unknown"
                is_master = false
                address = host[0]
                hostname = host[1]
                unless mastersConfig["master_servers"].nil?
                  unless mastersConfig["master_servers"]["hosts"].nil?
                    mastersConfig["master_servers"]["hosts"].each do |ms|
                      ms = ms.split()
                      unless ms.length < 2
                        if ms[1].eql?(hostname)
                          is_master = true
                          break
                        end
                      end
                    end
                  end
                end
                h = Host.find_by(hostname: hostname)
                if h.nil?
                  h = Host.new
                  h.version = version
                  h.is_master = is_master
                  h.hostname = hostname
                  h.address = address
                  h.save
                else 
                  unless h.address.eql?(address)
                    h.address = address
                    change = true 
                  end
                  unless h.is_master.eql?(is_master)
                    h.is_master = is_master
                    change = true 
                  end
                  h.save if change
                end
                m = Master.find_by(hostname: hostname)
                if is_master
                  if m.nil?
                    m = Master.new
                    m.hostname = hostname
                    m.vector = "{}"
                    m.save
                  end
                else
                  unless m.nil?
                    m.delete
                  end
                end
              end
            end
          end
        end
      end
    end
    
    unless mastersConfig["master_servers"].nil?
      unless mastersConfig["master_servers"]["hosts"].nil?
        mastersConfig["master_servers"]["hosts"].each do |host|
          host = host.split()
          unless host.length < 2
            version = "unknown"
            is_master = true
            address = host[0]
            hostname = host[1]
            h = Host.find_by(hostname: hostname)
            if h.nil?
              h = Host.new
              h.version = version
              h.is_master = is_master
              h.hostname = hostname
              h.address = address
              h.save
            else 
              change = false
              unless h.address.eql?(address)
                h.address = address
                change = true 
              end
              unless h.is_master.eql?(is_master)
                h.is_master = is_master
                change = true 
              end
              h.save if change
            end
            m = Master.find_by(hostname: hostname)
            if is_master
              if m.nil?
                m = Master.new
                m.hostname = hostname
                m.vector = "{}"
                m.save
              end
            else
              unless m.nil?
                m.delete
              end
            end
          end
        end
      end
    end
  end
end
