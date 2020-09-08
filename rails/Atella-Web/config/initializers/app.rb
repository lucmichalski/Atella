require 'yaml'
require 'redis'

include ApplicationHelper
if File.exist? Rails.root.join('config', 'app.yml')
    Rails.application.config.atella = YAML::load_file(Rails.root.join('config', 'app.yml'))
else
    Rails.application.config.atella = {}
end

