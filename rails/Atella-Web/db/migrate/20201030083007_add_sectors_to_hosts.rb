class AddSectorsToHosts < ActiveRecord::Migration[6.0]
  def change
    add_column :hosts, :sectors, :text
  end
end
