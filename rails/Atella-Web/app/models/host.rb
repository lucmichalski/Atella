class Host < ApplicationRecord

  def wrap_version(code)
    res = false
    self.version = "unknown"
    if code.nil?
      STDERR.print("[ERROR]: Code is nil!\n")
      return res
    end
    begin
      s = TCPSocket.open(self.address, 5223)
      s.puts("auth #{code}")
      loop do
        line = s.gets
        break if line.nil?
        l = line.rstrip.split
        case (l[0])
          when "+OK"
            break if l.length < 2
            case (l[1]) 
              when "bye!"
                break
              when "ack"
                break if l.length < 3
                case (l[2])
                  when "auth"
                    s.puts("get version")
                  when "master"
                    break if l.length < 4
                    status["status"] = JSON.parse(l[3])
                  when "version"
                    break if l.length < 4
                    self.version = l[3]
                    unless self.version.nil?
                      self.version.rstrip!  
                      res = true
                    end
                    s.puts("exit")
                  else
                    break
                end
              else
                break
            end
          else
            break
        end
      end
      s.close
    rescue => v
      STDERR.print("[ERROR]: #{v}\n")
      res = false
    end
    return res
  end

  def Host.reload_hosts_config
    settings = Rails.application.config.atella
    redis = Redis.new(url: "redis://#{ENV.fetch("REDIS_URL") { "127.0.0.1" }}")
    err =  nil
    masterConfigFlag = true
    hostsList = Hash.new

    begin
      mastersConfig = TOML.load_file(settings["atella"]["masterServersConfig"])
      sectorsConfig = TOML.load_file(settings["atella"]["sectorsConfig"])
    rescue => v
      err =  v
    end

    if err
      STDERR.print("[FATAL]: #{err}\n")  
      return
    end

    if mastersConfig["master_servers"].nil? && mastersConfig["master_servers"]["hosts"].nil?
      STDERR.print("[ERROR]: Incorrect master config!\n")  
      masterConfigFlag = false
    end

    if sectorsConfig["sectors"].nil?
      STDERR.print("[FATAL]: sectors config not present!\n")  
      return
    end
    
    sectorsConfig["sectors"].each do |sec|
      if sec.length < 2
        STDERR.print("[ERROR]: sector incorrect [#{sec}]!\n")
        next
      end
      if sec[1]["hosts"].nil?
        STDERR.print("[ERROR]: Not enouth hosts in sector [#{sec}]!\n")
        next
      end
      sec[1]["hosts"].each do |host|
        _host = host.split()
        if _host.length < 2
          STDERR.print("[ERROR]: Incorrect host [#{host}]!\n")
          next
        end
        hostsList[_host[1]] = Hash.new if hostsList[_host[1]].nil?
        hostsList[_host[1]][:version] = "unknown"
        hostsList[_host[1]][:is_master] = false
        hostsList[_host[1]][:address] = _host[0]
        hostsList[_host[1]][:hostname] = _host[1]
        hostsList[_host[1]][:sectors] = Array.new if hostsList[_host[1]][:sectors].nil?
        hostsList[_host[1]][:sectors] << sec[0]
      end
    end

    if masterConfigFlag
      mastersConfig["master_servers"]["hosts"].each do |host|
        _host = host.split()
        if _host.length < 2
          STDERR.print("[ERROR]: Incorrect host [#{host}]!\n")
          next
        end
        
        hostsList[_host[1]] = Hash.new if hostsList[_host[1]].nil?
        hostsList[_host[1]][:version] = "unknown"
        hostsList[_host[1]][:is_master] = true
        hostsList[_host[1]][:address] = _host[0]
        hostsList[_host[1]][:hostname] = _host[1]
        hostsList[_host[1]][:sectors] = Array.new if hostsList[_host[1]][:sectors].nil?
      end
    end

    hostsList.each do |v|
      _change = false
      _host = v[0]
      _st = v[1] 
      h = Host.find_by(hostname: _host)
      if h.nil?
        h = Host.new
        h.version = _st[:version]
        h.is_master = _st[:is_master]
        h.hostname = _st[:hostname]
        h.address = _st[:address]
        h.save
      else 
        unless h.address.eql?(_st[:address])
          h.address = _st[:address]
          change = true 
        end
        unless h.sectors.eql?(_st[:sectors])
          h.sectors = _st[:sectors]
          change = true 
        end
        unless h.is_master.eql?(_st[:is_master])
          h.is_master = _st[:is_master]
          change = true 
        end
        h.save if change
      end
    end
  end
end
