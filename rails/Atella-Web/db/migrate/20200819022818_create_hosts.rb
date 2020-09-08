class CreateHosts < ActiveRecord::Migration[6.0]
  def change
    create_table :hosts do |t|
      t.string :address, default: "unknown"
      t.string :hostname, default: "unknown"
      t.string :version, default: "unknown"

      t.timestamps
    end
  end
end
