export namespace config {
	
	export class Settings {
	    username: string;
	    password: string;
	    role: string;
	    isp: string;
	    remember: boolean;
	    autoStart: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.username = source["username"];
	        this.password = source["password"];
	        this.role = source["role"];
	        this.isp = source["isp"];
	        this.remember = source["remember"];
	        this.autoStart = source["autoStart"];
	    }
	}

}

export namespace main {
	
	export class AppState {
	    status: string;
	    message: string;
	    ssid: string;
	    interface: string;
	    ip: string;
	    mac: string;
	    signal: string;
	    account: string;
	    lastChecked: string;
	
	    static createFrom(source: any = {}) {
	        return new AppState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.ssid = source["ssid"];
	        this.interface = source["interface"];
	        this.ip = source["ip"];
	        this.mac = source["mac"];
	        this.signal = source["signal"];
	        this.account = source["account"];
	        this.lastChecked = source["lastChecked"];
	    }
	}

}

